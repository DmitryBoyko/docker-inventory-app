package app

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/parity"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// ExportService builds downloadable inventory snapshots (PS replacement).
type ExportService struct {
	Store *store.Store
}

// ExportFormat is json or csv.
type ExportFormat string

const (
	ExportJSON ExportFormat = "json"
	ExportCSV  ExportFormat = "csv"
)

// ExportScope selects which tables to include.
type ExportScope string

const (
	ExportScopeAll        ExportScope = "all"
	ExportScopeContainers ExportScope = "containers"
	ExportScopeStacks     ExportScope = "stacks"
)

// ExportResult is a file payload.
type ExportResult struct {
	ContentType        string
	ContentDisposition string
	Body               []byte
}

// Export builds a downloadable artifact from the current snapshot.
func (s *ExportService) Export(format, scope string) (ExportResult, error) {
	f, err := parseFormat(format)
	if err != nil {
		return ExportResult{}, err
	}
	sc, err := parseScope(scope)
	if err != nil {
		return ExportResult{}, err
	}
	snap := parity.FromStore(s.Store, "go")
	ts := time.Now().UTC().Format("20060102-150405")

	switch f {
	case ExportJSON:
		body, err := marshalJSON(snap, sc)
		if err != nil {
			return ExportResult{}, err
		}
		name := fmt.Sprintf("docker-visualizer-%s.json", ts)
		return ExportResult{
			ContentType:        "application/json; charset=utf-8",
			ContentDisposition: `attachment; filename="` + name + `"`,
			Body:               body,
		}, nil
	case ExportCSV:
		body, nameSuffix, err := marshalCSV(snap, sc)
		if err != nil {
			return ExportResult{}, err
		}
		name := fmt.Sprintf("docker-visualizer-%s-%s.csv", nameSuffix, ts)
		return ExportResult{
			ContentType:        "text/csv; charset=utf-8",
			ContentDisposition: `attachment; filename="` + name + `"`,
			Body:               body,
		}, nil
	default:
		return ExportResult{}, fmt.Errorf("unsupported format")
	}
}

func parseFormat(s string) (ExportFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "json":
		return ExportJSON, nil
	case "csv":
		return ExportCSV, nil
	default:
		return "", fmt.Errorf("format must be json or csv")
	}
}

func parseScope(s string) (ExportScope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all":
		return ExportScopeAll, nil
	case "containers":
		return ExportScopeContainers, nil
	case "stacks":
		return ExportScopeStacks, nil
	default:
		return "", fmt.Errorf("scope must be all, containers, or stacks")
	}
}

func marshalJSON(snap parity.Snapshot, scope ExportScope) ([]byte, error) {
	switch scope {
	case ExportScopeContainers:
		return json.MarshalIndent(map[string]any{
			"schemaVersion": snap.SchemaVersion,
			"source":        snap.Source,
			"capturedAt":    snap.CapturedAt,
			"containers":    snap.Containers,
		}, "", "  ")
	case ExportScopeStacks:
		return json.MarshalIndent(map[string]any{
			"schemaVersion": snap.SchemaVersion,
			"source":        snap.Source,
			"capturedAt":    snap.CapturedAt,
			"stacks":        snap.Stacks,
			"totals":        snap.Totals,
		}, "", "  ")
	default:
		return json.MarshalIndent(snap, "", "  ")
	}
}

func marshalCSV(snap parity.Snapshot, scope ExportScope) ([]byte, string, error) {
	switch scope {
	case ExportScopeStacks:
		b, err := stacksCSV(snap.Stacks)
		return b, "stacks", err
	case ExportScopeAll, ExportScopeContainers:
		b, err := containersCSV(snap.Containers)
		return b, "containers", err
	default:
		return nil, "", fmt.Errorf("unsupported scope")
	}
}

func containersCSV(rows []parity.ContainerRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"idShort", "name", "stack", "service", "state", "health", "restartCount",
		"writableLayerBytes", "volumeNames", "cpuPercent", "memoryBytes", "portExposures",
	})
	for _, r := range rows {
		_ = w.Write([]string{
			r.IDShort,
			r.Name,
			r.Stack,
			r.Service,
			r.State,
			r.Health,
			strconv.Itoa(r.RestartCount),
			fmtOptInt64(r.WritableLayerBytes),
			strings.Join(r.VolumeNames, ";"),
			formatFloat(r.CPUPercent),
			strconv.FormatInt(r.MemoryBytes, 10),
			strings.Join(r.PortExposures, ";"),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func stacksCSV(rows []parity.StackRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"name", "containerCount", "runningCount", "unhealthyCount", "restartedCount",
		"cpuPercent", "memoryBytes", "writableLayerBytes", "volumeNames", "volumeBytes",
	})
	for _, r := range rows {
		_ = w.Write([]string{
			r.Name,
			strconv.Itoa(r.ContainerCount),
			strconv.Itoa(r.RunningCount),
			strconv.Itoa(r.UnhealthyCount),
			strconv.Itoa(r.RestartedCount),
			formatFloat(r.CPUPercent),
			strconv.FormatInt(r.MemoryBytes, 10),
			fmtOptInt64(r.WritableLayerBytes),
			strings.Join(r.VolumeNames, ";"),
			fmtOptInt64(r.VolumeBytes),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func fmtOptInt64(p *int64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatInt(*p, 10)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}
