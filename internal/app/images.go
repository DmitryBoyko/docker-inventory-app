package app

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// ImagesService reads images from the snapshot.
type ImagesService struct {
	Store *store.Store
}

// ImagesResult is a list response.
type ImagesResult struct {
	Images      []domain.Image
	SnapshotAt  time.Time
	SnapshotAge time.Duration
}

// List returns images (optional q / dangling filter).
// dangling: "" | "true" | "false"
func (s *ImagesService) List(q, dangling string) ImagesResult {
	snap := s.Store.Load()
	out := make([]domain.Image, 0, len(snap.Images))
	ql := strings.ToLower(q)
	for _, img := range snap.Images {
		switch dangling {
		case "true":
			if !img.Dangling {
				continue
			}
		case "false":
			if img.Dangling {
				continue
			}
		}
		if ql != "" && !imageMatches(img, ql) {
			continue
		}
		out = append(out, img)
	}
	return ImagesResult{
		Images:      out,
		SnapshotAt:  snap.CollectedAt,
		SnapshotAge: snap.Age(),
	}
}

// Get returns one image by id, short id, or repo tag.
func (s *ImagesService) Get(idOrTag string) (*domain.Image, time.Time, bool) {
	snap := s.Store.Load()
	img, ok := snap.GetImage(idOrTag)
	if !ok {
		// Fallback: prefix match on id.
		for i := range snap.Images {
			if strings.HasPrefix(snap.Images[i].ID, idOrTag) || strings.HasPrefix(snap.Images[i].IDShort, idOrTag) {
				cp := snap.Images[i]
				return &cp, snap.CollectedAt, true
			}
		}
		return nil, snap.CollectedAt, false
	}
	cp := *img
	return &cp, snap.CollectedAt, true
}

func imageMatches(img domain.Image, q string) bool {
	if strings.Contains(strings.ToLower(img.ID), q) || strings.Contains(strings.ToLower(img.IDShort), q) {
		return true
	}
	for _, t := range img.RepoTags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	for _, d := range img.RepoDigests {
		if strings.Contains(strings.ToLower(d), q) {
			return true
		}
	}
	for _, c := range img.Containers {
		if strings.Contains(strings.ToLower(c), q) {
			return true
		}
	}
	return false
}
