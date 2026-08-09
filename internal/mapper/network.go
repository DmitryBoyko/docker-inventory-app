package mapper

import (
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/network"
)

// FromNetworkSummary maps Engine network list item → domain.
func FromNetworkSummary(n network.Summary) domain.Network {
	labels := cloneLabels(n.Labels)
	return domain.Network{
		ID:         n.ID,
		IDShort:    domain.ShortID(n.ID),
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Attachable: n.Attachable,
		Ingress:    n.Ingress,
		Labels:     labels,
	}
}
