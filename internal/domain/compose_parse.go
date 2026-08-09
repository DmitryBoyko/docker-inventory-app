package domain

import "strconv"

// ParseComposeLabels derives stack/service identity from container labels (ADR-008).
// Parity with scripts/docker-stack-inventory.ps1:
//   missing project → "standalone"; missing service → nil (UI may show "-").
func ParseComposeLabels(labels map[string]string) ComposeMeta {
	if labels == nil {
		return ComposeMeta{Project: StandaloneStack}
	}
	project := labels[LabelComposeProject]
	if project == "" {
		return ComposeMeta{Project: StandaloneStack}
	}
	meta := ComposeMeta{Project: project}
	if svc := labels[LabelComposeService]; svc != "" {
		s := svc
		meta.Service = &s
	}
	if num := labels[LabelComposeContainerNumber]; num != "" {
		if v, err := strconv.Atoi(num); err == nil {
			meta.ContainerNumber = &v
		}
	}
	return meta
}
