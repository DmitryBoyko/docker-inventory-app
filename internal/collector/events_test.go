package collector

import "testing"

func TestInterestingContainerAction(t *testing.T) {
	cases := map[string]bool{
		"start":                    true,
		"health_status: healthy":   true,
		"exec_start: /bin/sh":      false,
		"attach":                   false,
		"die":                      true,
	}
	for a, want := range cases {
		if got := interestingContainerAction(a); got != want {
			t.Fatalf("%s: got %v want %v", a, got, want)
		}
	}
}

func TestInterestingNetworkAction(t *testing.T) {
	if !interestingNetworkAction("connect") {
		t.Fatal("connect")
	}
	if interestingNetworkAction("attach") {
		t.Fatal("attach")
	}
}
