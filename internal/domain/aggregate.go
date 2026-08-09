package domain

// SumByteMetrics aggregates ByteMetrics with honest partial/unknown semantics (ADR-011).
func SumByteMetrics(items []ByteMetric) AggregateBytes {
	if len(items) == 0 {
		z := int64(0)
		return AggregateBytes{Bytes: &z, Available: true, Partial: false, UnknownCount: 0}
	}
	var sum int64
	unknown := 0
	for _, m := range items {
		if !m.Available || m.Bytes == nil {
			unknown++
			continue
		}
		sum += *m.Bytes
	}
	if unknown == len(items) {
		return AggregateBytes{
			Bytes:        nil,
			Available:    false,
			Partial:      true,
			UnknownCount: unknown,
			Reason:       ReasonUnsupported,
		}
	}
	out := sum
	return AggregateBytes{
		Bytes:        &out,
		Available:    true,
		Partial:      unknown > 0,
		UnknownCount: unknown,
	}
}

// SumUniqueVolumeUsage sums usage for unique volume names (PowerShell grand total semantics,
// but unknown sizes are not coerced to 0).
func SumUniqueVolumeUsage(names []string, byName map[string]Volume) AggregateBytes {
	seen := make(map[string]struct{}, len(names))
	var metrics []ByteMetric
	for _, n := range names {
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		v, ok := byName[n]
		if !ok {
			metrics = append(metrics, UnavailableBytes(ReasonDaemonOmitted))
			continue
		}
		metrics = append(metrics, v.Usage.ByteMetric)
	}
	return SumByteMetrics(metrics)
}

// FormatAnonVolumeName shortens 64-char anonymous volume IDs (PS Format-VolumeName).
func FormatAnonVolumeName(name string) string {
	if len(name) == 64 && isHex(name) {
		return "anon:" + name[:12] + "..."
	}
	return name
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
