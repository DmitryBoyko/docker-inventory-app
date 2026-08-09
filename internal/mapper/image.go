package mapper

import (
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/image"
)

// FromImageSummary maps Engine image list item → domain.
func FromImageSummary(img image.Summary) domain.Image {
	var created *time.Time
	if img.Created > 0 {
		t := time.Unix(img.Created, 0).UTC()
		created = &t
	}
	var shared *int64
	if img.SharedSize >= 0 {
		v := img.SharedSize
		shared = &v
	}
	tags := append([]string(nil), img.RepoTags...)
	digests := append([]string(nil), img.RepoDigests...)
	dangling := len(tags) == 0
	for _, t := range tags {
		if t == "<none>:<none>" {
			dangling = true
			break
		}
	}
	return domain.Image{
		ID:              img.ID,
		IDShort:         domain.ShortID(img.ID),
		RepoTags:        tags,
		RepoDigests:     digests,
		SizeBytes:       img.Size,
		SharedSizeBytes: shared,
		CreatedAt:       created,
		Labels:          cloneLabels(img.Labels),
		Dangling:        dangling,
	}
}
