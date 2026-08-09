package snapshots

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

// CaptureOptions identify the Docker host for a snapshot.
type CaptureOptions struct {
	HostName      string
	Endpoint      string
	Context       string
	DockerVersion string
	Label         string
	ID            string
	Now           time.Time
}

// Capture builds a sanitized Snapshot from live domain inventory (no env/secrets).
func Capture(
	containers []domain.Container,
	images []domain.Image,
	networks []domain.Network,
	volumes []domain.Volume,
	stacks []domain.Stack,
	opts CaptureOptions,
) Snapshot {
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	id := opts.ID
	if id == "" {
		id = now.Format("20060102T150405.000Z")
	}

	cv := make([]ContainerView, 0, len(containers))
	for _, c := range containers {
		svc := ""
		if c.Service != nil {
			svc = *c.Service
		}
		view := ContainerView{
			ID: c.ID, IDShort: c.IDShort, Name: c.Name, Stack: c.Stack, Service: svc,
			Image: c.Image, State: c.State, Health: c.Health, RestartCount: c.RestartCount,
			Labels: filterLabels(c.Labels),
		}
		if c.WritableLayer.Available && c.WritableLayer.Bytes != nil {
			b := *c.WritableLayer.Bytes
			view.WritableBytes = &b
		}
		if c.Stats != nil {
			mem := c.Stats.MemoryBytes
			cpu := c.Stats.CPUPercent
			view.MemoryBytes = &mem
			view.CPUPercent = &cpu
		}
		cv = append(cv, view)
	}

	iv := make([]ImageView, 0, len(images))
	for _, img := range images {
		iv = append(iv, ImageView{
			ID: img.ID, IDShort: img.IDShort, RepoTags: append([]string(nil), img.RepoTags...),
			SizeBytes: img.SizeBytes, ContainerCount: img.ContainerCount, Dangling: img.Dangling,
		})
	}

	nv := make([]NetworkView, 0, len(networks))
	for _, n := range networks {
		nv = append(nv, NetworkView{
			ID: n.ID, IDShort: n.IDShort, Name: n.Name, Driver: n.Driver, Internal: n.Internal,
			Containers: append([]string(nil), n.Containers...),
		})
	}

	vv := make([]VolumeView, 0, len(volumes))
	for _, v := range volumes {
		view := VolumeView{
			Name: v.Name, Driver: v.Driver, UsageAvailable: v.Usage.Available,
			Containers: append([]string(nil), v.Containers...),
			Stacks: append([]string(nil), v.Stacks...),
		}
		if v.Usage.Available && v.Usage.Bytes != nil {
			b := *v.Usage.Bytes
			view.UsageBytes = &b
		}
		vv = append(vv, view)
	}

	sv := make([]StackView, 0, len(stacks))
	for _, s := range stacks {
		sv = append(sv, StackView{
			Name: s.Name, ContainerCount: s.Resources.ContainerCount, RunningCount: s.Resources.RunningCount,
		})
	}

	return Snapshot{
		Meta: Meta{
			ID: id, CreatedAt: now, HostName: opts.HostName, Endpoint: opts.Endpoint,
			Context: opts.Context, DockerVersion: opts.DockerVersion, Label: opts.Label,
			Counts: Counts{
				Containers: len(cv), Images: len(iv), Networks: len(nv), Volumes: len(vv), Stacks: len(sv),
			},
		},
		Containers: cv, Images: iv, Networks: nv, Volumes: vv, Stacks: sv,
	}
}

func filterLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range in {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "com.docker.compose.") ||
			strings.HasPrefix(lk, "com.docker.stack.") ||
			lk == "org.opencontainers.image.title" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
