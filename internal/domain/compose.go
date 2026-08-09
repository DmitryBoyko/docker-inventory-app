package domain

// Compose label keys used for stack aggregation (ADR-008).
// Must stay aligned with scripts/docker-stack-inventory.ps1.
const (
	LabelComposeProject         = "com.docker.compose.project"
	LabelComposeService         = "com.docker.compose.service"
	LabelComposeContainerNumber = "com.docker.compose.container-number"
	LabelComposeVersion         = "com.docker.compose.version"
	LabelComposeWorkingDir      = "com.docker.compose.project.working_dir"
	LabelComposeConfigFiles     = "com.docker.compose.project.config_files"

	// StandaloneStack is the bucket for containers without a compose project label.
	StandaloneStack = "standalone"
)

// ComposeMeta is derived compose identity for a container.
type ComposeMeta struct {
	Project         string
	Service         *string
	ContainerNumber *int
}
