package domain

// TopConsumer is a ranked resource consumer (PowerShell "Top RAM").
type TopConsumer struct {
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	MemoryBytes   int64  `json:"memoryBytes"`
}

// Service groups containers within a stack by compose service name.
type Service struct {
	Name       string         `json:"name"`
	Stack      string         `json:"stack"`
	Containers []ContainerRef `json:"containers"`
}

// Stack is a virtual aggregate derived from Compose labels (ADR-008).
type Stack struct {
	Name            string          `json:"name"`
	Containers      []ContainerRef  `json:"containers"`
	Services        []Service       `json:"services,omitempty"`
	Resources       ResourceSummary `json:"resources"`
	VolumeNames     []string        `json:"volumeNames,omitempty"`
	VolumeUsage     AggregateBytes  `json:"volumeUsage"`
	UnhealthyCount  int             `json:"unhealthyCount"`
	RestartedCount  int             `json:"restartedCount"` // restartCount > 0 (PS parity)
	TopRAM          []TopConsumer   `json:"topRam,omitempty"`
}
