// Package provenance documents where UI values come from (Engine API + transforms).
package provenance

// Spec explains the origin of a displayed field.
type Spec struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	TitleKey        string   `json:"titleKey"`
	EntityKind      string   `json:"entityKind"`
	APIEndpoint     string   `json:"apiEndpoint"`
	DockerField     string   `json:"dockerField,omitempty"`
	Transformation  string   `json:"transformation,omitempty"`
	TransformationKey string `json:"transformationKey,omitempty"`
	Description     string   `json:"description"`
	DescriptionKey  string   `json:"descriptionKey"`
	RelatedCommands []string `json:"relatedCommands,omitempty"`
	Chain           []string `json:"chain,omitempty"` // human-readable pipeline steps
}

// Catalog returns all known provenance specs.
func Catalog() []Spec {
	return append([]Spec(nil), specs...)
}

// Get returns a spec by id.
func Get(id string) (Spec, bool) {
	for _, s := range specs {
		if s.ID == id {
			return s, true
		}
	}
	return Spec{}, false
}

var specs = []Spec{
	{
		ID: "container.cpuPercent", Title: "CPU percent", TitleKey: "prov.container.cpu.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/stats",
		DockerField: "cpu_stats / precpu_stats (usage & system delta)",
		Transformation: "delta CPU usage / delta system CPU × online CPUs × 100 (Docker CLI compatible)",
		TransformationKey: "prov.container.cpu.transform",
		Description: "Live CPU utilization sampled from the Engine stats stream and normalized by the visualizer.",
		DescriptionKey: "prov.container.cpu.desc",
		RelatedCommands: []string{"container.stats"},
		Chain: []string{"Docker Engine API stats", "cpu_stats delta", "normalize to percent", "UI"},
	},
	{
		ID: "container.memoryBytes", Title: "Memory usage", TitleKey: "prov.container.memory.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/stats",
		DockerField: "memory_stats.usage (minus inactive_file when present)",
		Transformation: "bytes → human-readable units",
		TransformationKey: "prov.bytes.transform",
		Description: "Container memory usage from Engine stats, adjusted like docker stats.",
		DescriptionKey: "prov.container.memory.desc",
		RelatedCommands: []string{"container.stats"},
		Chain: []string{"Docker Engine API stats", "memory_stats.usage", "bytes → display", "UI"},
	},
	{
		ID: "container.networkIO", Title: "Network I/O", TitleKey: "prov.container.net.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/stats",
		DockerField: "networks.*.rx_bytes / tx_bytes (sum)",
		Transformation: "sum interfaces → cumulative bytes → human-readable",
		TransformationKey: "prov.bytes.transform",
		Description: "Cumulative network receive/transmit bytes since container start.",
		DescriptionKey: "prov.container.net.desc",
		RelatedCommands: []string{"container.stats"},
	},
	{
		ID: "container.blockIO", Title: "Block I/O", TitleKey: "prov.container.block.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/stats",
		DockerField: "blkio_stats.io_service_bytes_recursive",
		Transformation: "sum Read/Write ops → bytes → human-readable",
		TransformationKey: "prov.bytes.transform",
		Description: "Cumulative block device read/write bytes.",
		DescriptionKey: "prov.container.block.desc",
		RelatedCommands: []string{"container.stats"},
	},
	{
		ID: "container.restartCount", Title: "Restart count", TitleKey: "prov.container.restarts.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/json",
		DockerField: "RestartCount",
		Transformation: "passthrough integer",
		TransformationKey: "prov.passthrough",
		Description: "Number of times the container has been restarted by Docker.",
		DescriptionKey: "prov.container.restarts.desc",
		RelatedCommands: []string{"container.inspect", "container.logs"},
	},
	{
		ID: "container.health", Title: "Health", TitleKey: "prov.container.health.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/json",
		DockerField: "State.Health.Status",
		Transformation: "normalize to healthy|unhealthy|starting|none",
		TransformationKey: "prov.container.health.transform",
		Description: "Healthcheck status reported by the Docker daemon.",
		DescriptionKey: "prov.container.health.desc",
		RelatedCommands: []string{"container.inspect", "container.logs"},
	},
	{
		ID: "container.writableLayer", Title: "Writable layer size", TitleKey: "prov.container.writable.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/json (SizeRw) / system df",
		DockerField: "SizeRw",
		Transformation: "bytes → human-readable; may be unavailable",
		TransformationKey: "prov.bytes.transform",
		Description: "Size of the container writable (upper) layer when the daemon provides it.",
		DescriptionKey: "prov.container.writable.desc",
		RelatedCommands: []string{"container.diff", "system.df"},
		Chain: []string{"Container inspect SizeRw or system df", "ByteMetric availability", "UI"},
	},
	{
		ID: "container.ip", Title: "Container IP", TitleKey: "prov.container.ip.title",
		EntityKind: "container",
		APIEndpoint: "GET /containers/{id}/json",
		DockerField: "NetworkSettings.Networks.*.IPAddress",
		Transformation: "map endpoints to NetworkEndpoint list",
		TransformationKey: "prov.passthrough",
		Description: "IP address assigned on each attached Docker network.",
		DescriptionKey: "prov.container.ip.desc",
		RelatedCommands: []string{"container.inspect", "network.inspect"},
	},
	{
		ID: "volume.size", Title: "Volume size", TitleKey: "prov.volume.size.title",
		EntityKind: "volume",
		APIEndpoint: "GET /system/df",
		DockerField: "Volumes[].UsageData.Size",
		Transformation: "bytes → human-readable; unavailable when daemon omits usage",
		TransformationKey: "prov.bytes.transform",
		Description: "Volume disk usage from docker system df, linked back to inventory volumes.",
		DescriptionKey: "prov.volume.size.desc",
		RelatedCommands: []string{"system.df", "volume.inspect"},
		Chain: []string{"docker system df", "Volume usage", "volume association", "stack aggregation", "UI"},
	},
	{
		ID: "image.size", Title: "Image size", TitleKey: "prov.image.size.title",
		EntityKind: "image",
		APIEndpoint: "GET /images/json",
		DockerField: "Size",
		Transformation: "bytes → human-readable",
		TransformationKey: "prov.bytes.transform",
		Description: "Image size as reported by the Engine image list API.",
		DescriptionKey: "prov.image.size.desc",
		RelatedCommands: []string{"image.inspect", "system.df"},
	},
}
