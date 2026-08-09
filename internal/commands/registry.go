package commands

// Registry returns the built-in command definitions.
func Registry() []Definition {
	return append([]Definition(nil), definitions...)
}

// Lookup returns a definition by id.
func Lookup(id string) (Definition, bool) {
	for _, d := range definitions {
		if d.ID == id {
			return d, true
		}
	}
	return Definition{}, false
}

// ForEntity returns definitions applicable to a kind.
func ForEntity(kind EntityKind) []Definition {
	out := make([]Definition, 0)
	for _, d := range definitions {
		if d.EntityKind == kind {
			out = append(out, d)
		}
	}
	return out
}

var definitions = []Definition{
	{
		ID: "container.inspect", TitleKey: "cmd.container.inspect.title", DescriptionKey: "cmd.container.inspect.desc",
		Title: "Inspect container", Description: "Shows low-level Docker configuration and runtime state of the container, including mounts, networks, environment, runtime configuration and metadata.",
		Category: "inspect", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"inspect", "{{ref}}"},
	},
	{
		ID: "container.stats", TitleKey: "cmd.container.stats.title", DescriptionKey: "cmd.container.stats.desc",
		Title: "Show statistics", Description: "Streams live resource usage statistics for the container (CPU, memory, network and block I/O).",
		Category: "stats", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"stats", "{{ref}}", "--no-stream"},
	},
	{
		ID: "container.logs", TitleKey: "cmd.container.logs.title", DescriptionKey: "cmd.container.logs.desc",
		Title: "Show logs", Description: "Fetches stdout/stderr logs from the container. Useful when investigating crashes and restart loops.",
		Category: "logs", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"logs", "{{ref}}", "--tail", "200"},
	},
	{
		ID: "container.logs.tail500", TitleKey: "cmd.container.logs.tail500.title", DescriptionKey: "cmd.container.logs.tail500.desc",
		Title: "Show recent logs (500)", Description: "Fetches the last 500 log lines — a deeper window for diagnosing repeated failures.",
		Category: "logs", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"logs", "{{ref}}", "--tail", "500"},
	},
	{
		ID: "container.top", TitleKey: "cmd.container.top.title", DescriptionKey: "cmd.container.top.desc",
		Title: "Show processes", Description: "Lists processes running inside the container (docker top).",
		Category: "runtime", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"top", "{{ref}}"},
	},
	{
		ID: "container.diff", TitleKey: "cmd.container.diff.title", DescriptionKey: "cmd.container.diff.desc",
		Title: "Show filesystem changes", Description: "Shows files added, changed or deleted in the container writable layer relative to the image.",
		Category: "filesystem", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"diff", "{{ref}}"},
	},
	{
		ID: "container.port", TitleKey: "cmd.container.port.title", DescriptionKey: "cmd.container.port.desc",
		Title: "Show published ports", Description: "Lists port mappings published by the container.",
		Category: "network", EntityKind: EntityContainer, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"port", "{{ref}}"},
	},
	{
		ID: "container.exec", TitleKey: "cmd.container.exec.title", DescriptionKey: "cmd.container.exec.desc",
		Title: "Open interactive shell", Description: "Starts an interactive shell inside a running container. Requires a TTY and is interactive.",
		Category: "exec", EntityKind: EntityContainer, RiskLevel: RiskInteractive, RequiresTTY: true, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"exec", "-it", "{{ref}}", "sh"},
	},
	{
		ID: "network.inspect", TitleKey: "cmd.network.inspect.title", DescriptionKey: "cmd.network.inspect.desc",
		Title: "Inspect network", Description: "Shows Docker network configuration, options, IPAM and connected containers.",
		Category: "inspect", EntityKind: EntityNetwork, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"network", "inspect", "{{ref}}"},
	},
	{
		ID: "volume.inspect", TitleKey: "cmd.volume.inspect.title", DescriptionKey: "cmd.volume.inspect.desc",
		Title: "Inspect volume", Description: "Shows volume driver, mountpoint, labels and options.",
		Category: "inspect", EntityKind: EntityVolume, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"volume", "inspect", "{{ref}}"},
	},
	{
		ID: "image.inspect", TitleKey: "cmd.image.inspect.title", DescriptionKey: "cmd.image.inspect.desc",
		Title: "Inspect image", Description: "Shows image configuration, layers metadata, labels and size details.",
		Category: "inspect", EntityKind: EntityImage, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"image", "inspect", "{{ref}}"},
	},
	{
		ID: "image.history", TitleKey: "cmd.image.history.title", DescriptionKey: "cmd.image.history.desc",
		Title: "Show image history", Description: "Lists the image layer history (created-by commands and sizes).",
		Category: "inspect", EntityKind: EntityImage, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"image", "history", "{{ref}}"},
	},
	{
		ID: "system.df", TitleKey: "cmd.system.df.title", DescriptionKey: "cmd.system.df.desc",
		Title: "Disk usage", Description: "Shows Docker disk usage for images, containers, volumes and build cache.",
		Category: "system", EntityKind: EntitySystem, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"system", "df"},
	},
	{
		ID: "system.info", TitleKey: "cmd.system.info.title", DescriptionKey: "cmd.system.info.desc",
		Title: "Engine info", Description: "Displays Docker daemon system-wide information.",
		Category: "system", EntityKind: EntitySystem, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"info"},
	},
	{
		ID: "system.version", TitleKey: "cmd.system.version.title", DescriptionKey: "cmd.system.version.desc",
		Title: "Client/server version", Description: "Shows Docker client and server version details.",
		Category: "system", EntityKind: EntitySystem, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"version"},
	},
	{
		ID: "stack.ps", TitleKey: "cmd.stack.ps.title", DescriptionKey: "cmd.stack.ps.desc",
		Title: "List stack containers", Description: "Lists containers belonging to a Compose project/stack by label filter.",
		Category: "stack", EntityKind: EntityStack, RiskLevel: RiskReadOnly, RequiresDockerCLI: true,
		SupportsBash: true, SupportsPowerShell: true, SupportsCMD: true,
		ArgsTemplate: []string{"ps", "--filter", "label=com.docker.compose.project={{ref}}"},
	},
}
