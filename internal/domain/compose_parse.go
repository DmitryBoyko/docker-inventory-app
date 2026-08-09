package domain

import (
	"strconv"
	"strings"
)

// ParseComposeLabels derives stack/service identity from container labels (ADR-008).
//
// Precedence:
//  1. Compose project/service (PowerShell parity)
//  2. Else Swarm stack namespace (+ optional swarm service name)
//  3. Else standalone
//
// Missing service → nil (UI may show "-").
func ParseComposeLabels(labels map[string]string) ComposeMeta {
	if labels == nil {
		return ComposeMeta{Project: StandaloneStack, Source: "standalone"}
	}

	if project := strings.TrimSpace(labels[LabelComposeProject]); project != "" {
		meta := ComposeMeta{Project: project, Source: "compose"}
		if svc := strings.TrimSpace(labels[LabelComposeService]); svc != "" {
			s := svc
			meta.Service = &s
		}
		if num := strings.TrimSpace(labels[LabelComposeContainerNumber]); num != "" {
			if v, err := strconv.Atoi(num); err == nil {
				meta.ContainerNumber = &v
			}
		}
		return meta
	}

	if ns := strings.TrimSpace(labels[LabelSwarmStackNamespace]); ns != "" {
		meta := ComposeMeta{Project: ns, Source: "swarm"}
		if svc := swarmServiceShort(ns, labels[LabelSwarmServiceName]); svc != "" {
			s := svc
			meta.Service = &s
		}
		return meta
	}

	return ComposeMeta{Project: StandaloneStack, Source: "standalone"}
}

// swarmServiceShort turns "mystack_web" + namespace "mystack" into "web".
func swarmServiceShort(namespace, fullName string) string {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return ""
	}
	prefix := namespace + "_"
	if strings.HasPrefix(fullName, prefix) && len(fullName) > len(prefix) {
		return fullName[len(prefix):]
	}
	return fullName
}
