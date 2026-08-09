package domain

import "testing"

func TestAvailableBytes(t *testing.T) {
	m := AvailableBytes(42)
	if !m.Available || m.Bytes == nil || *m.Bytes != 42 {
		t.Fatalf("%+v", m)
	}
}

func TestUnavailableBytes(t *testing.T) {
	m := UnavailableBytes(ReasonUnsupported)
	if m.Available || m.Bytes != nil || m.Reason != ReasonUnsupported {
		t.Fatalf("%+v", m)
	}
}
