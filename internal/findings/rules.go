package findings

import (
	"fmt"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

func analyzeContainers(containers []domain.Container, th Thresholds) []Finding {
	out := make([]Finding, 0)
	for _, c := range containers {
		ent := EntityRef{Kind: "container", ID: c.ID, Name: c.Name}

		if c.State == domain.ContainerStateRestarting || c.RestartCount >= th.RestartCountCritical {
			sev := SeverityCritical
			if c.State != domain.ContainerStateRestarting && c.RestartCount < th.RestartCountCritical {
				sev = SeverityWarning
			}
			out = append(out, Finding{
				ID: fmt.Sprintf("container.restarts:%s", c.IDShort), RuleID: "container.restarts",
				Severity: sev, Entity: ent,
				TitleKey: "finding.container.restarts.title", DescriptionKey: "finding.container.restarts.desc",
				ReasonKey: "finding.container.restarts.reason", RecommendationKey: "finding.container.restarts.rec",
				Title: fmt.Sprintf("Container %s has restarted %d times", c.Name, c.RestartCount),
				Description: "Repeated restarts usually indicate a crash loop, failing dependency, or misconfiguration.",
				Reason: fmt.Sprintf("RestartCount=%d (warning≥%d, critical≥%d); state=%s", c.RestartCount, th.RestartCountWarning, th.RestartCountCritical, c.State),
				Recommendation: "Inspect the container and review recent logs.",
				Evidence: map[string]any{"restartCount": c.RestartCount, "state": string(c.State), "thresholdCritical": th.RestartCountCritical},
				RelatedCommands: []string{"container.inspect", "container.logs.tail500"},
			})
		} else if c.RestartCount >= th.RestartCountWarning {
			out = append(out, Finding{
				ID: fmt.Sprintf("container.restarts.warn:%s", c.IDShort), RuleID: "container.restarts",
				Severity: SeverityWarning, Entity: ent,
				TitleKey: "finding.container.restarts.title", DescriptionKey: "finding.container.restarts.desc",
				ReasonKey: "finding.container.restarts.reason", RecommendationKey: "finding.container.restarts.rec",
				Title: fmt.Sprintf("Container %s has restarted %d times", c.Name, c.RestartCount),
				Description: "Elevated restart count may indicate instability.",
				Reason: fmt.Sprintf("RestartCount=%d ≥ warning threshold %d", c.RestartCount, th.RestartCountWarning),
				Recommendation: "Check logs and health status.",
				Evidence: map[string]any{"restartCount": c.RestartCount, "thresholdWarning": th.RestartCountWarning},
				RelatedCommands: []string{"container.inspect", "container.logs"},
			})
		}

		if c.Health == domain.HealthUnhealthy {
			out = append(out, Finding{
				ID: fmt.Sprintf("container.unhealthy:%s", c.IDShort), RuleID: "container.unhealthy",
				Severity: SeverityCritical, Entity: ent,
				TitleKey: "finding.container.unhealthy.title", DescriptionKey: "finding.container.unhealthy.desc",
				ReasonKey: "finding.container.unhealthy.reason", RecommendationKey: "finding.container.unhealthy.rec",
				Title: fmt.Sprintf("Container %s is unhealthy", c.Name),
				Description: "The Docker healthcheck is currently failing.",
				Reason: "State.Health.Status = unhealthy",
				Recommendation: "Inspect healthcheck configuration and application logs.",
				Evidence: map[string]any{"health": string(c.Health)},
				RelatedCommands: []string{"container.inspect", "container.logs"},
			})
		}

		if c.State == domain.ContainerStateExited || c.State == domain.ContainerStateDead {
			out = append(out, Finding{
				ID: fmt.Sprintf("container.stopped:%s", c.IDShort), RuleID: "container.stopped",
				Severity: SeverityInfo, Entity: ent,
				TitleKey: "finding.container.stopped.title", DescriptionKey: "finding.container.stopped.desc",
				ReasonKey: "finding.container.stopped.reason", RecommendationKey: "finding.container.stopped.rec",
				Title: fmt.Sprintf("Container %s is stopped", c.Name),
				Description: "The container is not running.",
				Reason: fmt.Sprintf("state=%s status=%s", c.State, c.Status),
				Recommendation: "Confirm whether this is expected; inspect exit status if not.",
				Evidence: map[string]any{"state": string(c.State), "status": c.Status},
				RelatedCommands: []string{"container.inspect", "container.logs"},
			})
		}

		if c.Health == domain.HealthNone && c.State == domain.ContainerStateRunning {
			out = append(out, Finding{
				ID: fmt.Sprintf("container.no_healthcheck:%s", c.IDShort), RuleID: "container.no_healthcheck",
				Severity: SeverityInfo, Entity: ent,
				TitleKey: "finding.container.no_healthcheck.title", DescriptionKey: "finding.container.no_healthcheck.desc",
				ReasonKey: "finding.container.no_healthcheck.reason", RecommendationKey: "finding.container.no_healthcheck.rec",
				Title: fmt.Sprintf("Container %s has no healthcheck", c.Name),
				Description: "Without a healthcheck Docker cannot report application readiness.",
				Reason: "Health = none on a running container",
				Recommendation: "Consider adding a HEALTHCHECK to the image or Compose service.",
				Evidence: map[string]any{"health": string(c.Health)},
				RelatedCommands: []string{"container.inspect"},
			})
		}

		if c.WritableLayer.Available && c.WritableLayer.Bytes != nil {
			sz := *c.WritableLayer.Bytes
			if sz >= th.WritableLayerCritBytes || sz >= th.WritableLayerWarnBytes {
				sev := SeverityWarning
				if sz >= th.WritableLayerCritBytes {
					sev = SeverityCritical
				}
				out = append(out, Finding{
					ID: fmt.Sprintf("container.writable:%s", c.IDShort), RuleID: "container.writable_layer",
					Severity: sev, Entity: ent,
					TitleKey: "finding.container.writable.title", DescriptionKey: "finding.container.writable.desc",
					ReasonKey: "finding.container.writable.reason", RecommendationKey: "finding.container.writable.rec",
					Title: fmt.Sprintf("Container %s has a large writable layer", c.Name),
					Description: "A large writable layer often means logs or data written inside the container filesystem.",
					Reason: fmt.Sprintf("SizeRw=%d bytes (warn≥%d, crit≥%d)", sz, th.WritableLayerWarnBytes, th.WritableLayerCritBytes),
					Recommendation: "Investigate filesystem changes with docker diff and move data to volumes when appropriate.",
					Evidence: map[string]any{"writableLayerBytes": sz, "thresholdWarn": th.WritableLayerWarnBytes, "thresholdCrit": th.WritableLayerCritBytes},
					RelatedCommands: []string{"container.diff", "container.inspect"},
				})
			}
		}

		if c.Stats != nil && c.State == domain.ContainerStateRunning {
			if c.Stats.MemoryPercent >= th.MemoryWarnPercent && c.Stats.MemoryLimitBytes > 0 {
				out = append(out, Finding{
					ID: fmt.Sprintf("container.memory:%s", c.IDShort), RuleID: "container.high_memory",
					Severity: SeverityWarning, Entity: ent,
					TitleKey: "finding.container.memory.title", DescriptionKey: "finding.container.memory.desc",
					ReasonKey: "finding.container.memory.reason", RecommendationKey: "finding.container.memory.rec",
					Title: fmt.Sprintf("Container %s memory usage is high", c.Name),
					Description: "Memory utilization is close to the container limit.",
					Reason: fmt.Sprintf("memoryPercent=%.1f ≥ %.1f", c.Stats.MemoryPercent, th.MemoryWarnPercent),
					Recommendation: "Inspect process list and consider raising limits or reducing usage.",
					Evidence: map[string]any{"memoryPercent": c.Stats.MemoryPercent, "memoryBytes": c.Stats.MemoryBytes, "limit": c.Stats.MemoryLimitBytes},
					RelatedCommands: []string{"container.stats", "container.top"},
				})
			}
			if c.Stats.CPUPercent >= th.CPUWarnPercent {
				out = append(out, Finding{
					ID: fmt.Sprintf("container.cpu:%s", c.IDShort), RuleID: "container.high_cpu",
					Severity: SeverityWarning, Entity: ent,
					TitleKey: "finding.container.cpu.title", DescriptionKey: "finding.container.cpu.desc",
					ReasonKey: "finding.container.cpu.reason", RecommendationKey: "finding.container.cpu.rec",
					Title: fmt.Sprintf("Container %s CPU usage is high", c.Name),
					Description: "CPU utilization is unusually high for a single sample window.",
					Reason: fmt.Sprintf("cpuPercent=%.1f ≥ %.1f", c.Stats.CPUPercent, th.CPUWarnPercent),
					Recommendation: "Check top processes and application load.",
					Evidence: map[string]any{"cpuPercent": c.Stats.CPUPercent},
					RelatedCommands: []string{"container.stats", "container.top"},
				})
			}
		}
	}
	return out
}

func analyzeImages(images []domain.Image, th Thresholds) []Finding {
	out := make([]Finding, 0)
	for _, img := range images {
		name := img.IDShort
		if len(img.RepoTags) > 0 {
			name = img.RepoTags[0]
		}
		ent := EntityRef{Kind: "image", ID: img.ID, Name: name}
		if img.Dangling {
			out = append(out, Finding{
				ID: fmt.Sprintf("image.dangling:%s", img.IDShort), RuleID: "image.dangling",
				Severity: SeverityInfo, Entity: ent,
				TitleKey: "finding.image.dangling.title", DescriptionKey: "finding.image.dangling.desc",
				ReasonKey: "finding.image.dangling.reason", RecommendationKey: "finding.image.dangling.rec",
				Title: fmt.Sprintf("Dangling image %s", img.IDShort),
				Description: "Image has no tags (dangling) and may be leftover from rebuilds.",
				Reason: "dangling=true (no RepoTags)",
				Recommendation: "Remove unused dangling images after confirming they are not needed.",
				Evidence: map[string]any{"id": img.ID, "sizeBytes": img.SizeBytes},
				RelatedCommands: []string{"image.inspect", "system.df"},
			})
		}
		if img.ContainerCount == 0 && !img.Dangling {
			out = append(out, Finding{
				ID: fmt.Sprintf("image.unused:%s", img.IDShort), RuleID: "image.unused",
				Severity: SeverityInfo, Entity: ent,
				TitleKey: "finding.image.unused.title", DescriptionKey: "finding.image.unused.desc",
				ReasonKey: "finding.image.unused.reason", RecommendationKey: "finding.image.unused.rec",
				Title: fmt.Sprintf("Unused image %s", name),
				Description: "No containers in the current inventory reference this image.",
				Reason: "containerCount=0",
				Recommendation: "Consider removing unused images to reclaim disk.",
				Evidence: map[string]any{"containerCount": 0, "sizeBytes": img.SizeBytes},
				RelatedCommands: []string{"image.inspect", "system.df"},
			})
		}
		if img.SizeBytes >= th.ImageLargeBytes {
			out = append(out, Finding{
				ID: fmt.Sprintf("image.large:%s", img.IDShort), RuleID: "image.large",
				Severity: SeverityWarning, Entity: ent,
				TitleKey: "finding.image.large.title", DescriptionKey: "finding.image.large.desc",
				ReasonKey: "finding.image.large.reason", RecommendationKey: "finding.image.large.rec",
				Title: fmt.Sprintf("Large image %s", name),
				Description: "Image size exceeds the configured large-image threshold.",
				Reason: fmt.Sprintf("sizeBytes=%d ≥ %d", img.SizeBytes, th.ImageLargeBytes),
				Recommendation: "Review image layers and multi-stage builds.",
				Evidence: map[string]any{"sizeBytes": img.SizeBytes, "threshold": th.ImageLargeBytes},
				RelatedCommands: []string{"image.history", "image.inspect"},
			})
		}
	}
	return out
}

func analyzeVolumes(volumes []domain.Volume, th Thresholds) []Finding {
	out := make([]Finding, 0)
	for _, v := range volumes {
		ent := EntityRef{Kind: "volume", ID: v.Name, Name: v.Name}
		if len(v.Containers) == 0 {
			out = append(out, Finding{
				ID: fmt.Sprintf("volume.unused:%s", v.Name), RuleID: "volume.unused",
				Severity: SeverityInfo, Entity: ent,
				TitleKey: "finding.volume.unused.title", DescriptionKey: "finding.volume.unused.desc",
				ReasonKey: "finding.volume.unused.reason", RecommendationKey: "finding.volume.unused.rec",
				Title: fmt.Sprintf("Unused volume %s", v.Name),
				Description: "No containers currently mount this volume.",
				Reason: "containers linked = 0",
				Recommendation: "Confirm the volume is obsolete before removing it.",
				Evidence: map[string]any{"driver": v.Driver},
				RelatedCommands: []string{"volume.inspect", "system.df"},
			})
		}
		if !v.Usage.Available {
			out = append(out, Finding{
				ID: fmt.Sprintf("volume.size_unavailable:%s", v.Name), RuleID: "volume.size_unavailable",
				Severity: SeverityInfo, Entity: ent,
				TitleKey: "finding.volume.size_unavailable.title", DescriptionKey: "finding.volume.size_unavailable.desc",
				ReasonKey: "finding.volume.size_unavailable.reason", RecommendationKey: "finding.volume.size_unavailable.rec",
				Title: fmt.Sprintf("Volume size unavailable for %s", v.Name),
				Description: "Docker did not provide usage data for this volume.",
				Reason: fmt.Sprintf("usage.available=false reason=%s", v.Usage.Reason),
				Recommendation: "Size requires docker system df support for the volume driver.",
				Evidence: map[string]any{"reason": v.Usage.Reason, "driver": v.Driver},
				RelatedCommands: []string{"system.df", "volume.inspect"},
			})
		} else if v.Usage.Bytes != nil && *v.Usage.Bytes >= th.VolumeLargeBytes {
			out = append(out, Finding{
				ID: fmt.Sprintf("volume.large:%s", v.Name), RuleID: "volume.large",
				Severity: SeverityWarning, Entity: ent,
				TitleKey: "finding.volume.large.title", DescriptionKey: "finding.volume.large.desc",
				ReasonKey: "finding.volume.large.reason", RecommendationKey: "finding.volume.large.rec",
				Title: fmt.Sprintf("Large volume %s", v.Name),
				Description: "Volume usage exceeds the configured threshold.",
				Reason: fmt.Sprintf("usageBytes=%d ≥ %d", *v.Usage.Bytes, th.VolumeLargeBytes),
				Recommendation: "Inspect data growth and retention policies.",
				Evidence: map[string]any{"usageBytes": *v.Usage.Bytes, "threshold": th.VolumeLargeBytes},
				RelatedCommands: []string{"volume.inspect", "system.df"},
			})
		}
	}
	return out
}

func analyzeNetworks(networks []domain.Network) []Finding {
	out := make([]Finding, 0)
	for _, n := range networks {
		// Skip built-in networks that are expected to exist unused.
		if n.Name == "bridge" || n.Name == "host" || n.Name == "none" {
			continue
		}
		if len(n.Containers) == 0 {
			out = append(out, Finding{
				ID: fmt.Sprintf("network.unused:%s", n.IDShort), RuleID: "network.unused",
				Severity: SeverityInfo, Entity: EntityRef{Kind: "network", ID: n.ID, Name: n.Name},
				TitleKey: "finding.network.unused.title", DescriptionKey: "finding.network.unused.desc",
				ReasonKey: "finding.network.unused.reason", RecommendationKey: "finding.network.unused.rec",
				Title: fmt.Sprintf("Unused network %s", n.Name),
				Description: "No containers are attached to this network.",
				Reason: "containers linked = 0",
				Recommendation: "Remove obsolete user-defined networks to reduce clutter.",
				Evidence: map[string]any{"driver": n.Driver, "internal": n.Internal},
				RelatedCommands: []string{"network.inspect"},
			})
		}
	}
	return out
}
