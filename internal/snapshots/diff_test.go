package snapshots

import (
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

func TestCaptureOmitsNonComposeLabels(t *testing.T) {
	c := domain.Container{
		ID: "1", IDShort: "1", Name: "api", Image: "api:1", State: domain.ContainerStateRunning,
		Labels: map[string]string{
			"com.docker.compose.project": "demo",
			"secret":                     "should-not-persist",
			"password":                   "nope",
		},
	}
	snap := Capture([]domain.Container{c}, nil, nil, nil, nil, CaptureOptions{HostName: "default", ID: "t1", Now: time.Unix(0, 0).UTC()})
	if snap.Containers[0].Labels["secret"] != "" {
		t.Fatal("secret label persisted")
	}
	if snap.Containers[0].Labels["com.docker.compose.project"] != "demo" {
		t.Fatal("compose label missing")
	}
}

func TestDiffAddedRemovedModified(t *testing.T) {
	left := Snapshot{
		Meta: Meta{ID: "a", CreatedAt: time.Unix(1, 0).UTC()},
		Containers: []ContainerView{
			{ID: "1", Name: "old-api", Image: "api:1", State: domain.ContainerStateRunning},
			{ID: "2", Name: "postgres", Image: "pg:15", State: domain.ContainerStateRunning, RestartCount: 0},
		},
		Volumes: []VolumeView{{Name: "postgres-data", UsageAvailable: true, UsageBytes: int64Ptr(1 << 30)}},
	}
	right := Snapshot{
		Meta: Meta{ID: "b", CreatedAt: time.Unix(2, 0).UTC()},
		Containers: []ContainerView{
			{ID: "3", Name: "api-2", Image: "api:2", State: domain.ContainerStateRunning},
			{ID: "2", Name: "postgres", Image: "pg:16", State: domain.ContainerStateRunning, RestartCount: 1},
		},
		Volumes: []VolumeView{{Name: "postgres-data", UsageAvailable: true, UsageBytes: int64Ptr(5<<30 + 200<<20)}},
	}
	d := Compare(left, right)
	if !hasChange(d.Containers, ChangeAdded, "api-2") {
		t.Fatalf("missing added: %+v", d.Containers)
	}
	if !hasChange(d.Containers, ChangeRemoved, "old-api") {
		t.Fatalf("missing removed: %+v", d.Containers)
	}
	if !hasChange(d.Containers, ChangeModified, "postgres") {
		t.Fatalf("missing modified postgres: %+v", d.Containers)
	}
	if !hasChange(d.Volumes, ChangeModified, "postgres-data") {
		t.Fatalf("missing volume change: %+v", d.Volumes)
	}
}

func TestDiffEmptyToPopulated(t *testing.T) {
	left := Snapshot{Meta: Meta{ID: "empty"}}
	right := Snapshot{
		Meta: Meta{ID: "full"},
		Containers: []ContainerView{{ID: "1", Name: "c1"}},
		Images:     []ImageView{{ID: "i1", IDShort: "i1"}},
	}
	d := Compare(left, right)
	if len(d.Containers) != 1 || d.Containers[0].Kind != ChangeAdded {
		t.Fatalf("%+v", d.Containers)
	}
	if len(d.Images) != 1 || d.Images[0].Kind != ChangeAdded {
		t.Fatalf("%+v", d.Images)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap := Snapshot{Meta: Meta{ID: "20260101T000000.000Z", CreatedAt: time.Unix(0, 0).UTC(), HostName: "default"}, Containers: []ContainerView{{ID: "1", Name: "n"}}}
	if err := st.Save(snap); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(snap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Containers[0].Name != "n" {
		t.Fatalf("%+v", got)
	}
	list, err := st.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

func hasChange(cs []Change, kind ChangeKind, name string) bool {
	for _, c := range cs {
		if c.Kind == kind && c.Name == name {
			return true
		}
	}
	return false
}

func int64Ptr(v int64) *int64 { return &v }
