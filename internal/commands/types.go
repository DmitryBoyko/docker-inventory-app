// Package commands is the Docker CLI Command Registry (generate + explain only; no execution).
package commands

// RiskLevel classifies operational risk of a CLI command.
type RiskLevel string

const (
	RiskReadOnly      RiskLevel = "READ_ONLY"
	RiskInteractive   RiskLevel = "INTERACTIVE"
	RiskStateChanging RiskLevel = "STATE_CHANGING"
	RiskDestructive   RiskLevel = "DESTRUCTIVE"
)

// Shell is a target shell for rendering.
type Shell string

const (
	ShellBash       Shell = "bash"
	ShellPowerShell Shell = "powershell"
	ShellCMD        Shell = "cmd"
)

// EntityKind groups commands by Docker object type.
type EntityKind string

const (
	EntityContainer EntityKind = "container"
	EntityNetwork   EntityKind = "network"
	EntityVolume    EntityKind = "volume"
	EntityImage     EntityKind = "image"
	EntitySystem    EntityKind = "system"
	EntityStack     EntityKind = "stack"
)

// Definition is a reusable command template (not a rendered string).
type Definition struct {
	ID                string     `json:"id"`
	TitleKey          string     `json:"titleKey"`
	DescriptionKey    string     `json:"descriptionKey"`
	Title             string     `json:"title"`             // English fallback
	Description       string     `json:"description"`       // English fallback
	Category          string     `json:"category"`
	EntityKind        EntityKind `json:"entityKind"`
	RiskLevel         RiskLevel  `json:"riskLevel"`
	RequiresTTY       bool       `json:"requiresTTY"`
	RequiresDockerCLI bool       `json:"requiresDockerCLI"`
	SupportsBash      bool       `json:"supportsBash"`
	SupportsPowerShell bool      `json:"supportsPowerShell"`
	SupportsCMD       bool       `json:"supportsCMD"`
	// ArgsTemplate is docker subcommand tokens after global flags, with {{ref}} placeholder.
	ArgsTemplate []string `json:"-"`
}

// ConnectionContext describes how the visualizer reached the Engine (for CLI global flags).
type ConnectionContext struct {
	HostName string // registry host name (ADR-014)
	Endpoint string // Engine URL e.g. unix:///var/run/docker.sock or tcp://...
	Source   string // explicit|docker_host|context|default
	Context  string // docker context name when Source=context
}

// Target identifies the entity for which commands are generated.
type Target struct {
	Kind EntityKind
	// Ref is the CLI-friendly reference (container name, volume name, network name/id, image ref).
	Ref string
	// ID is optional full id (used when Ref empty).
	ID string
}

// Rendered is a concrete, copy-ready command for one shell.
type Rendered struct {
	DefinitionID   string    `json:"definitionId"`
	TitleKey       string    `json:"titleKey"`
	DescriptionKey string    `json:"descriptionKey"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	EntityKind     EntityKind `json:"entityKind"`
	RiskLevel      RiskLevel `json:"riskLevel"`
	RequiresTTY    bool      `json:"requiresTTY"`
	Shell          Shell     `json:"shell"`
	Command        string    `json:"command"`
	EntityRef      string    `json:"entityRef"`
}
