package domain

import (
	"sort"
	"strings"
	"time"
)

// Image is a Docker image summary for the UI/API.
type Image struct {
	ID              string            `json:"id"`
	IDShort         string            `json:"idShort"`
	RepoTags        []string          `json:"repoTags,omitempty"`
	RepoDigests     []string          `json:"repoDigests,omitempty"`
	SizeBytes       int64             `json:"sizeBytes"`
	SharedSizeBytes *int64            `json:"sharedSizeBytes"` // null when daemon omitted (-1)
	CreatedAt       *time.Time        `json:"createdAt,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Containers      []string          `json:"containers,omitempty"` // names linked from inventory
	ContainerCount  int               `json:"containerCount"`       // linked count (authoritative for UI)
	Dangling        bool              `json:"dangling"`
}

// LinkImages attaches reverse container refs by ImageID / RepoTags / short image name.
func LinkImages(images []Image, containers []Container) []Image {
	hits := map[string]map[string]struct{}{} // key → container names
	add := func(key, name string) {
		if key == "" || name == "" {
			return
		}
		m, ok := hits[key]
		if !ok {
			m = map[string]struct{}{}
			hits[key] = m
		}
		m[name] = struct{}{}
	}

	for _, c := range containers {
		add(c.ImageID, c.Name)
		add(ShortID(c.ImageID), c.Name)
		add(c.Image, c.Name)
	}

	out := make([]Image, len(images))
	copy(out, images)
	for i := range out {
		names := map[string]struct{}{}
		merge := func(key string) {
			for n := range hits[key] {
				names[n] = struct{}{}
			}
		}
		merge(out[i].ID)
		merge(out[i].IDShort)
		for _, tag := range out[i].RepoTags {
			merge(tag)
			merge(ShortImage(tag))
		}
		out[i].Containers = sortedKeys(names)
		out[i].ContainerCount = len(out[i].Containers)
		out[i].Dangling = len(out[i].RepoTags) == 0 || (len(out[i].RepoTags) == 1 && out[i].RepoTags[0] == "<none>:<none>")
	}
	sort.Slice(out, func(i, j int) bool {
		ai := primaryTag(out[i])
		aj := primaryTag(out[j])
		if ai == aj {
			return out[i].ID < out[j].ID
		}
		return strings.ToLower(ai) < strings.ToLower(aj)
	})
	return out
}

func primaryTag(img Image) string {
	for _, t := range img.RepoTags {
		if t != "" && t != "<none>:<none>" {
			return t
		}
	}
	return img.IDShort
}
