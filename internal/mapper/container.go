package mapper

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
)

// FromSummary maps a ContainerList item into a domain Container (pre-inspect).
func FromSummary(s container.Summary) domain.Container {
	compose := domain.ParseComposeLabels(s.Labels)
	health := domain.HealthNone
	if s.Health != nil {
		health = domain.NormalizeHealth(string(s.Health.Status))
	}
	health = domain.ResolveHealth(health, s.Status)
	ports := mapPortSummaries(s.Ports)

	return domain.Container{
		ID:               s.ID,
		IDShort:          domain.ShortID(s.ID),
		Name:             domain.ContainerName(s.Names, ""),
		Stack:            compose.Project,
		Service:          compose.Service,
		ContainerNumber:  compose.ContainerNumber,
		Image:            domain.ShortImage(s.Image),
		ImageID:          s.ImageID,
		State:            domain.ContainerState(strings.ToLower(string(s.State))),
		Status:           s.Status,
		Health:           health,
		Ports:            ports,
		ExternalExposure: domain.BuildExternalExposure(ports),
		ExposureScope:    domain.SummarizeExposure(ports),
		Endpoints:        mapNetworkEndpoints(networkMapFromSummary(s.NetworkSettings)),
		Mounts:           mapMountPoints(s.Mounts),
		WritableLayer:    domain.AvailableBytes(s.SizeRw),
		Labels:           cloneLabels(s.Labels),
	}
}

// EnrichFromInspect merges inspect fields onto a container built from Summary.
func EnrichFromInspect(c *domain.Container, insp container.InspectResponse) {
	if c == nil {
		return
	}
	c.RestartCount = insp.RestartCount
	if insp.Name != "" {
		c.Name = domain.ContainerName(nil, insp.Name)
	}
	if insp.State != nil {
		if insp.State.Status != "" {
			c.State = domain.ContainerState(strings.ToLower(string(insp.State.Status)))
		}
		if insp.State.Health != nil {
			h := domain.NormalizeHealth(string(insp.State.Health.Status))
			c.Health = domain.ResolveHealth(h, c.Status)
		} else {
			c.Health = domain.ResolveHealth(c.Health, c.Status)
		}
		if t, ok := parseDockerTime(insp.State.StartedAt); ok {
			c.StartedAt = &t
			if c.State == domain.ContainerStateRunning {
				sec := int64(time.Since(t).Seconds())
				if sec < 0 {
					sec = 0
				}
				c.UptimeSeconds = &sec
			}
		}
		if t, ok := parseDockerTime(insp.State.FinishedAt); ok && t.Year() > 1 {
			c.FinishedAt = &t
		}
	}
	if len(insp.Mounts) > 0 {
		c.Mounts = mapMountPoints(insp.Mounts)
	}
	if insp.NetworkSettings != nil && len(insp.NetworkSettings.Networks) > 0 {
		c.Endpoints = mapNetworkEndpoints(insp.NetworkSettings.Networks)
	}
	if insp.SizeRw != nil {
		c.WritableLayer = domain.AvailableBytes(*insp.SizeRw)
	}
}

func mapPortSummaries(ports []container.PortSummary) []domain.Port {
	in := make([]domain.PortBindingInput, 0, len(ports))
	for _, p := range ports {
		row := domain.PortBindingInput{
			ContainerPort: p.PrivatePort,
			Protocol:      p.Type,
			HostPort:      p.PublicPort,
			Published:     p.PublicPort != 0,
		}
		if p.IP.IsValid() {
			row.HostIP = p.IP.String()
		}
		in = append(in, row)
	}
	return domain.MapPortBindings(in)
}

func mapMountPoints(mounts []container.MountPoint) []domain.Mount {
	out := make([]domain.Mount, 0, len(mounts))
	for _, m := range mounts {
		out = append(out, domain.Mount{
			Type:        mapMountType(m.Type),
			Name:        m.Name,
			Source:      m.Source,
			Destination: m.Destination,
			RW:          m.RW,
		})
	}
	return out
}

func mapMountType(t mount.Type) domain.MountType {
	switch t {
	case mount.TypeVolume:
		return domain.MountTypeVolume
	case mount.TypeBind:
		return domain.MountTypeBind
	case mount.TypeTmpfs:
		return domain.MountTypeTmpfs
	case mount.TypeNamedPipe:
		return domain.MountTypeNpipe
	default:
		return domain.MountType(t)
	}
}

func networkMapFromSummary(ns *container.NetworkSettingsSummary) map[string]*network.EndpointSettings {
	if ns == nil {
		return nil
	}
	return ns.Networks
}

func mapNetworkEndpoints(networks map[string]*network.EndpointSettings) []domain.NetworkEndpoint {
	if len(networks) == 0 {
		return nil
	}
	out := make([]domain.NetworkEndpoint, 0, len(networks))
	for name, ep := range networks {
		if ep == nil {
			continue
		}
		row := domain.NetworkEndpoint{
			NetworkID:   ep.NetworkID,
			NetworkName: name,
			MacAddress:  ep.MacAddress.String(),
		}
		if ep.IPAddress.IsValid() {
			row.IPAddress = ep.IPAddress.String()
		}
		if ep.Gateway.IsValid() {
			row.Gateway = ep.Gateway.String()
		}
		out = append(out, row)
	}
	return out
}

func cloneLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func parseDockerTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "0001-01-01") {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}
