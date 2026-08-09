package mapper

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/system"
)

// FromInfo maps Engine /info into domain.SystemInfo.
func FromInfo(info system.Info, apiVersion string) domain.SystemInfo {
	return domain.SystemInfo{
		ID:                info.ID,
		Name:              info.Name,
		ServerVersion:     info.ServerVersion,
		APIVersion:        apiVersion,
		OS:                info.OperatingSystem,
		OSVersion:         info.OSVersion,
		OSType:            info.OSType,
		Architecture:      info.Architecture,
		KernelVersion:     info.KernelVersion,
		NCPU:              info.NCPU,
		MemTotalBytes:     info.MemTotal,
		Driver:            info.Driver,
		DockerRootDir:     info.DockerRootDir,
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		Images:            info.Images,
		SystemTimeUTC:     systemTimeUTC(info.SystemTime),
	}
}

// systemTimeUTC normalizes Docker Info.SystemTime to UTC RFC3339.
func systemTimeUTC(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return ""
}
