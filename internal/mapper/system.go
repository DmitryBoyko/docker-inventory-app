package mapper

import (
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
	}
}
