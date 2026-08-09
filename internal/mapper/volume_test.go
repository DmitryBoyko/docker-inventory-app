package mapper

import (
	"testing"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/volume"
)

func TestFromVolume_SizeUnavailable(t *testing.T) {
	v := FromVolume(volume.Volume{
		Name:   "nfs-data",
		Driver: "local",
		UsageData: &volume.UsageData{
			Size:     -1,
			RefCount: 2,
		},
	})
	if v.Usage.Available || v.Usage.Bytes != nil {
		t.Fatalf("%+v", v.Usage)
	}
	if v.Usage.Links == nil || *v.Usage.Links != 2 {
		t.Fatalf("links=%v", v.Usage.Links)
	}
	if v.Usage.Reason != domain.ReasonUnsupported {
		t.Fatalf("reason=%s", v.Usage.Reason)
	}
}

func TestFromVolume_NotLocalDriver(t *testing.T) {
	v := FromVolume(volume.Volume{
		Name:   "x",
		Driver: "nfs",
		UsageData: &volume.UsageData{
			Size:     -1,
			RefCount: -1,
		},
	})
	if v.Usage.Reason != domain.ReasonNotLocalDriver {
		t.Fatalf("reason=%s", v.Usage.Reason)
	}
	if v.Usage.Links != nil {
		t.Fatalf("links should be null for -1")
	}
}

func TestMergeVolumeLists(t *testing.T) {
	listed := []volume.Volume{{Name: "a", Driver: "local"}}
	usage := []volume.Volume{{
		Name: "a", Driver: "local",
		UsageData: &volume.UsageData{Size: 42, RefCount: 1},
	}}
	out := MergeVolumeLists(listed, usage)
	if len(out) != 1 || !out[0].Usage.Available || out[0].Usage.Bytes == nil || *out[0].Usage.Bytes != 42 {
		t.Fatalf("%+v", out)
	}
}
