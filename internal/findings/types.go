// Package findings implements explainable Docker anomaly diagnostics (not the ops support dump).
package findings

import "github.com/epm-games/docker-visualizer/internal/domain"

// Severity of a finding.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

// EntityRef points at a Docker object in the UI.
type EntityRef struct {
	Kind string `json:"kind"` // container|image|volume|network|system|stack
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Finding is one explainable diagnostic result.
type Finding struct {
	ID              string    `json:"id"`
	RuleID          string    `json:"ruleId"`
	Severity        Severity  `json:"severity"`
	Entity          EntityRef `json:"entity"`
	TitleKey        string    `json:"titleKey"`
	DescriptionKey  string    `json:"descriptionKey"`
	ReasonKey       string    `json:"reasonKey"`
	RecommendationKey string  `json:"recommendationKey"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Reason          string    `json:"reason"`
	Recommendation  string    `json:"recommendation"`
	Evidence        map[string]any `json:"evidence,omitempty"`
	RelatedCommands []string  `json:"relatedCommands,omitempty"`
}

// Thresholds holds configurable detection constants (backend-owned).
type Thresholds struct {
	RestartCountCritical   int
	RestartCountWarning    int
	WritableLayerWarnBytes int64
	WritableLayerCritBytes int64
	MemoryWarnPercent      float64
	CPUWarnPercent         float64
	ImageLargeBytes        int64
	VolumeLargeBytes       int64
}

// DefaultThresholds returns sensible defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{
		RestartCountCritical:   10,
		RestartCountWarning:    3,
		WritableLayerWarnBytes: 2 << 30,  // 2 GiB
		WritableLayerCritBytes: 5 << 30,  // 5 GiB
		MemoryWarnPercent:      90,
		CPUWarnPercent:         90,
		ImageLargeBytes:        2 << 30,  // 2 GiB
		VolumeLargeBytes:       10 << 30, // 10 GiB
	}
}

// Analyze runs all rules against an inventory snapshot view.
func Analyze(containers []domain.Container, images []domain.Image, volumes []domain.Volume, networks []domain.Network, th Thresholds) []Finding {
	var out []Finding
	out = append(out, analyzeContainers(containers, th)...)
	out = append(out, analyzeImages(images, th)...)
	out = append(out, analyzeVolumes(volumes, th)...)
	out = append(out, analyzeNetworks(networks)...)
	return out
}
