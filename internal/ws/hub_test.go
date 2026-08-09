package ws

import (
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

func TestClientFilterStats(t *testing.T) {
	h := NewHub(nil)
	c := NewClient(h, nil, "default")
	c.subscribe(ChannelStats, &StatsFilters{Stack: "prod"})

	items := []StatsItem{
		{ID: "1", IDShort: "1", Name: "a", Stack: "prod", Stats: &domain.ContainerStats{CPUPercent: 1}},
		{ID: "2", IDShort: "2", Name: "b", Stack: "other", Stats: &domain.ContainerStats{CPUPercent: 2}},
	}
	env := Envelope{Type: TypeContainerStats, Data: items}
	if !c.accepts(env) {
		t.Fatal("expected accepts stats")
	}
	got := c.filterStats(env)
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("filter=%+v", got)
	}
}

func TestHubPublishDoesNotBlock(t *testing.T) {
	h := NewHub(nil)
	done := make(chan struct{})
	go func() {
		h.Run(done)
	}()
	defer close(done)

	for i := 0; i < 100; i++ {
		h.Publish(Envelope{Type: TypePing, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	}
}

func TestInterestingActions(t *testing.T) {
	// kept in collector; smoke Encode helper
	b, err := Encode(Envelope{Type: TypePing, Timestamp: "t"})
	if err != nil || len(b) == 0 {
		t.Fatal(err)
	}
}
