package mapper

import "testing"

func TestSystemTimeUTC(t *testing.T) {
	got := systemTimeUTC("2026-08-09T13:42:15.123456789+07:00")
	want := "2026-08-09T06:42:15Z"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if systemTimeUTC("") != "" {
		t.Fatal("empty")
	}
}
