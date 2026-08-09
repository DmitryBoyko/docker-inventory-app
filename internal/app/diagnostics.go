package app

import (
	"net"
	"runtime"
	"time"

	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/store"
	"github.com/epm-games/docker-visualizer/internal/uiembed"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// EventsStatus is the subset of EventsCollector needed for diagnostics.
type EventsStatus interface {
	Connected() bool
	LastError() string
}

// DiagnosticsService builds support snapshots (localhost-only via HTTP).
type DiagnosticsService struct {
	Store         *store.Store
	Docker        *docker.Client
	Hub           *ws.Hub
	Events        EventsStatus
	Health        *observability.Registry
	Version       string
	Commit        string
	Listen        string
	Intervals     map[string]string
	DockerTimeout string
	AuthEnabled   bool
	StartedAt     time.Time
}

// Diagnostics is the JSON payload for /system/diagnostics.
type Diagnostics struct {
	Timestamp       string                          `json:"timestamp"`
	Version         string                          `json:"version"`
	Commit          string                          `json:"commit"`
	GoVersion       string                          `json:"goVersion"`
	GOOS            string                          `json:"goos"`
	GOARCH          string                          `json:"goarch"`
	UptimeSeconds   int64                           `json:"uptimeSeconds"`
	Listen          string                          `json:"listen"`
	ListenLoopback  bool                            `json:"listenLoopback"`
	AuthEnabled     bool                            `json:"authEnabled"`
	ReadOnly        bool                            `json:"readOnly"`
	UIEmbedded      bool                            `json:"uiEmbedded"`
	Intervals       map[string]string               `json:"intervals"`
	Docker          any                             `json:"docker"`
	EventsConnected bool                            `json:"eventsConnected"`
	EventsError     string                          `json:"eventsError,omitempty"`
	WSClients       int                             `json:"wsClients"`
	Snapshot        SnapshotDiag                    `json:"snapshot"`
	Collectors      []observability.CollectorHealth `json:"collectors"`
	Runtime         RuntimeDiag                     `json:"runtime"`
}

// SnapshotDiag summarizes inventory freshness/size.
type SnapshotDiag struct {
	Version        uint64 `json:"version"`
	HasData        bool   `json:"hasData"`
	CollectedAt    string `json:"collectedAt,omitempty"`
	StatsAt        string `json:"statsAt,omitempty"`
	SystemAt       string `json:"systemAt,omitempty"`
	AgeMs          int64  `json:"ageMs"`
	CollectError   string `json:"collectError,omitempty"`
	ContainerCount int    `json:"containerCount"`
	RunningCount   int    `json:"runningCount"`
	NetworkCount   int    `json:"networkCount"`
	VolumeCount    int    `json:"volumeCount"`
	ImageCount     int    `json:"imageCount"`
}

// RuntimeDiag is process memory/goroutine info.
type RuntimeDiag struct {
	Goroutines int    `json:"goroutines"`
	AllocBytes uint64 `json:"allocBytes"`
	SysBytes   uint64 `json:"sysBytes"`
	NumGC      uint32 `json:"numGC"`
}

// Get builds the diagnostics payload.
func (s *DiagnosticsService) Get() Diagnostics {
	now := time.Now().UTC()
	snap := s.Store.Load()
	running := 0
	for _, c := range snap.Containers {
		if string(c.State) == "running" {
			running++
		}
	}
	var dockerStatus any
	if s.Docker != nil {
		dockerStatus = s.Docker.Status()
	}
	eventsOK := false
	eventsErr := ""
	if s.Events != nil {
		eventsOK = s.Events.Connected()
		eventsErr = s.Events.LastError()
	}
	wsClients := 0
	if s.Hub != nil {
		wsClients = s.Hub.ClientCount()
	}
	var collectors []observability.CollectorHealth
	if s.Health != nil {
		collectors = s.Health.Snapshot()
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	uptime := int64(0)
	if !s.StartedAt.IsZero() {
		uptime = int64(now.Sub(s.StartedAt).Seconds())
	}

	return Diagnostics{
		Timestamp:       now.Format(time.RFC3339Nano),
		Version:         s.Version,
		Commit:          s.Commit,
		GoVersion:       runtime.Version(),
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		UptimeSeconds:   uptime,
		Listen:          s.Listen,
		ListenLoopback:  isListenLoopback(s.Listen),
		AuthEnabled:     s.AuthEnabled,
		ReadOnly:        true,
		UIEmbedded:      uiembed.Available(),
		Intervals:       s.Intervals,
		Docker:          dockerStatus,
		EventsConnected: eventsOK,
		EventsError:     eventsErr,
		WSClients:       wsClients,
		Snapshot: SnapshotDiag{
			Version:        snap.Version,
			HasData:        snap.HasData(),
			CollectedAt:    formatOptTime(snap.CollectedAt),
			StatsAt:        formatOptTime(snap.StatsAt),
			SystemAt:       formatOptTime(snap.SystemAt),
			AgeMs:          snap.Age().Milliseconds(),
			CollectError:   snap.Err,
			ContainerCount: len(snap.Containers),
			RunningCount:   running,
			NetworkCount:   len(snap.Networks),
			VolumeCount:    len(snap.Volumes),
			ImageCount:     len(snap.Images),
		},
		Collectors: collectors,
		Runtime: RuntimeDiag{
			Goroutines: runtime.NumGoroutine(),
			AllocBytes: ms.Alloc,
			SysBytes:   ms.Sys,
			NumGC:      ms.NumGC,
		},
	}
}

func formatOptTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func isListenLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
}
