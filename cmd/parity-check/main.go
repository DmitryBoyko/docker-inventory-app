// Command parity-check compares a Go inventory snapshot to the PowerShell exporter.
//
//	go run ./cmd/parity-check
//	go run ./cmd/parity-check -skip-stats -ps-script scripts/docker-stack-inventory.ps1
//	go run ./cmd/parity-check -go-out go.json -ps-out ps.json
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/epm-games/docker-visualizer/internal/collector"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/parity"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	psScript := flag.String("ps-script", "scripts/docker-stack-inventory.ps1", "Path to PowerShell inventory script")
	goOut := flag.String("go-out", "", "Write Go parity JSON to this path")
	psOut := flag.String("ps-out", "", "Write PS parity JSON to this path (temp used if empty)")
	skipPS := flag.Bool("skip-ps", false, "Only collect/write Go snapshot (no compare)")
	skipStats := flag.Bool("skip-stats", false, "Ignore CPU/memory diffs (different sample windows)")
	psZero := flag.Bool("ps-volume-zero", true, "Treat missing volume/writable bytes as 0 (PS ?→0 mode)")
	timeout := flag.Duration("docker-timeout", 30*time.Second, "Docker request timeout")
	flag.Parse()

	cli, err := docker.Connect(docker.Options{Timeout: *timeout})
	if err != nil {
		log.Error("docker connect", "err", err)
		os.Exit(2)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	st := store.New()
	inv := &collector.InventoryCollector{Docker: cli, Store: st, Log: log}
	if err := inv.Refresh(ctx); err != nil {
		log.Warn("inventory refresh", "err", err)
	}
	stats := &collector.StatsCollector{Docker: cli, Store: st, Log: log}
	stats.Refresh(ctx)
	sys := &collector.SystemCollector{Docker: cli, Store: st, Log: log}
	sys.Refresh(ctx)

	goSnap := parity.FromStore(st, "go")
	if *goOut != "" {
		if err := parity.WriteFile(*goOut, goSnap); err != nil {
			log.Error("write go-out", "err", err)
			os.Exit(2)
		}
		log.Info("wrote go snapshot", "path", *goOut)
	}

	if *skipPS {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(goSnap)
		return
	}

	psPath := *psOut
	if psPath == "" {
		tmp, err := os.CreateTemp("", "parity-ps-*.json")
		if err != nil {
			log.Error("temp", "err", err)
			os.Exit(2)
		}
		psPath = tmp.Name()
		_ = tmp.Close()
		defer os.Remove(psPath)
	}

	if err := runPowerShellJSON(*psScript, psPath, log); err != nil {
		log.Error("powershell export", "err", err)
		os.Exit(2)
	}

	psSnap, err := parity.LoadFile(psPath)
	if err != nil {
		log.Error("load ps json", "err", err)
		os.Exit(2)
	}

	rep := parity.Compare(psSnap, goSnap, parity.Options{
		SkipStats:                     *skipStats,
		TreatMissingVolumeBytesAsZero: *psZero,
	})

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)

	if !rep.OK {
		log.Error("parity failed", "diffs", len(rep.Diffs))
		os.Exit(1)
	}
	log.Info("parity ok", "containers", rep.CandCount, "notes", rep.Notes)
}

func runPowerShellJSON(script, outPath string, log *slog.Logger) error {
	absScript, err := filepath.Abs(script)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absScript); err != nil {
		return fmt.Errorf("ps script: %w", err)
	}
	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	args := []string{"-NoProfile", "-File", absScript, "-JsonOut", absOut}
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell.exe", args...)
	default:
		// Prefer pwsh if present.
		if _, err := exec.LookPath("pwsh"); err == nil {
			cmd = exec.Command("pwsh", args...)
		} else {
			return fmt.Errorf("pwsh not found; install PowerShell 7 or use -skip-ps / compare JSON files manually")
		}
	}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	log.Info("running powershell exporter", "script", absScript, "out", absOut)
	return cmd.Run()
}
