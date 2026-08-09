package domain

import "testing"

func TestSumByteMetrics_AllAvailable(t *testing.T) {
	a := SumByteMetrics([]ByteMetric{AvailableBytes(10), AvailableBytes(5)})
	if !a.Available || a.Partial || a.Bytes == nil || *a.Bytes != 15 {
		t.Fatalf("%+v", a)
	}
}

func TestSumByteMetrics_Partial(t *testing.T) {
	a := SumByteMetrics([]ByteMetric{AvailableBytes(10), UnavailableBytes(ReasonUnsupported)})
	if !a.Available || !a.Partial || a.UnknownCount != 1 || a.Bytes == nil || *a.Bytes != 10 {
		t.Fatalf("%+v", a)
	}
}

func TestSumByteMetrics_AllUnknown(t *testing.T) {
	a := SumByteMetrics([]ByteMetric{UnavailableBytes(ReasonUnsupported)})
	if a.Available || a.Bytes != nil || !a.Partial {
		t.Fatalf("%+v", a)
	}
}

func TestFormatAnonVolumeName(t *testing.T) {
	anon := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if FormatAnonVolumeName(anon) != "anon:0123456789ab..." {
		t.Fatalf("%s", FormatAnonVolumeName(anon))
	}
	if FormatAnonVolumeName("webapp-data") != "webapp-data" {
		t.Fatal()
	}
}
