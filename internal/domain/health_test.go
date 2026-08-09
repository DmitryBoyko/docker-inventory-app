package domain

import "testing"

func TestHealthFromStatusFallback(t *testing.T) {
	cases := map[string]HealthState{
		"Up 3 minutes (healthy)":         HealthHealthy,
		"Up 1 second (unhealthy)":        HealthUnhealthy,
		"Up Less than a second (health: starting)": HealthStarting,
		"Up 10 seconds (starting)":       HealthStarting,
		"Exited (0) 2 days ago":          HealthNone,
		"Up 5 minutes":                   HealthNone,
	}
	for in, want := range cases {
		if got := HealthFromStatusFallback(in); got != want {
			t.Fatalf("%q: got %s want %s", in, got, want)
		}
	}
}

func TestResolveHealth(t *testing.T) {
	if got := ResolveHealth(HealthNone, "Up (healthy)"); got != HealthHealthy {
		t.Fatalf("got %s", got)
	}
	if got := ResolveHealth(HealthUnhealthy, "Up (healthy)"); got != HealthUnhealthy {
		t.Fatalf("primary must win: %s", got)
	}
}
