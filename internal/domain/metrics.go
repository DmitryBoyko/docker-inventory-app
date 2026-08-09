package domain

// ByteMetric represents a byte quantity that may be unavailable.
// Never treat unavailable as zero in aggregates (ADR-011).
type ByteMetric struct {
	Bytes     *int64 `json:"bytes"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// Known reasons for ByteMetric / AggregateBytes.
const (
	ReasonUnsupported    = "unsupported"
	ReasonNotLocalDriver = "not_local_driver"
	ReasonDaemonOmitted  = "daemon_omitted"
	ReasonCollectError   = "collect_error"
	ReasonPending        = "pending"
)

// AvailableBytes returns a present metric.
func AvailableBytes(v int64) ByteMetric {
	b := v
	return ByteMetric{Bytes: &b, Available: true}
}

// UnavailableBytes returns an absent metric with a reason.
func UnavailableBytes(reason string) ByteMetric {
	return ByteMetric{Bytes: nil, Available: false, Reason: reason}
}

// AggregateBytes summarizes a set of ByteMetrics.
type AggregateBytes struct {
	Bytes        *int64 `json:"bytes"`
	Available    bool   `json:"available"`
	Partial      bool   `json:"partial"`
	UnknownCount int    `json:"unknownCount"`
	Reason       string `json:"reason,omitempty"`
}

// ResourceSummary is stack/host-level resource rollup.
//
// Units:
//   - CPUPercent: Docker CLI-compatible percent (may exceed 100 on multi-core)
//   - MemoryBytes: bytes, running containers only when used as stack/host sum
//   - WritableLayer / VolumeData: ByteMetric / AggregateBytes semantics
type ResourceSummary struct {
	CPUPercent     float64        `json:"cpuPercent"`
	MemoryBytes    int64          `json:"memoryBytes"`
	WritableLayer  AggregateBytes `json:"writableLayer"`
	VolumeData     AggregateBytes `json:"volumeData"`
	ContainerCount int            `json:"containerCount"`
	RunningCount   int            `json:"runningCount"`
}
