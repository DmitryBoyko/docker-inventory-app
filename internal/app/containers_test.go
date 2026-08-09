package app

import (
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestListFilters(t *testing.T) {
	svcName := "web"
	st := store.New()
	st.Replace([]domain.Container{
		{ID: "aaa111111111", IDShort: "aaa111111111", Name: "a", Stack: "prod", Service: &svcName, State: domain.ContainerStateRunning, Health: domain.HealthHealthy, Image: "nginx"},
		{ID: "bbb222222222", IDShort: "bbb222222222", Name: "b", Stack: "standalone", State: domain.ContainerStateExited, Health: domain.HealthNone, Image: "redis"},
	}, time.Now().UTC(), "")

	svc := &ContainersService{Store: st}
	res := svc.List(ContainerQuery{Stack: "prod"})
	if len(res.Containers) != 1 || res.Containers[0].Name != "a" {
		t.Fatalf("%+v", res.Containers)
	}
	res = svc.List(ContainerQuery{Q: "redis"})
	if len(res.Containers) != 1 || res.Containers[0].Name != "b" {
		t.Fatalf("%+v", res.Containers)
	}
	c, _, ok := svc.Get("aaa111111111")
	if !ok || c.Name != "a" {
		t.Fatalf("get short failed")
	}
}
