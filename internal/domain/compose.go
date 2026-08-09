package domain

// Compose / Swarm label keys used for stack aggregation (ADR-008).
// Compose keys must stay aligned with scripts/docker-stack-inventory.ps1.
const (
	LabelComposeProject         = "com.docker.compose.project"
	LabelComposeService         = "com.docker.compose.service"
	LabelComposeContainerNumber = "com.docker.compose.container-number"
	LabelComposeVersion         = "com.docker.compose.version"
	LabelComposeWorkingDir      = "com.docker.compose.project.working_dir"
	LabelComposeConfigFiles     = "com.docker.compose.project.config_files"

	// Swarm stack deploy labels (V2).
	LabelSwarmStackNamespace = "com.docker.stack.namespace"
	LabelSwarmServiceName    = "com.docker.swarm.service.name"

	// StandaloneStack is the bucket for containers without a compose/swarm project label.
	StandaloneStack = "standalone"
)

// ComposeMeta is derived compose/swarm identity for a container.
type ComposeMeta struct {
	Project         string
	Service         *string
	ContainerNumber *int
	Source          string // "compose" | "swarm" | "standalone"
}
