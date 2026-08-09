package commands

import (
	"strings"

	"github.com/epm-games/docker-visualizer/internal/docker"
)

// Quote escapes a single argument for the given shell.
func Quote(shell Shell, arg string) string {
	switch shell {
	case ShellPowerShell:
		return quotePowerShell(arg)
	case ShellCMD:
		return quoteCMD(arg)
	default:
		return quoteBash(arg)
	}
}

func quoteBash(s string) string {
	if s == "" {
		return "''"
	}
	if !needsQuote(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func quotePowerShell(s string) string {
	if s == "" {
		return "''"
	}
	if !needsQuote(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func quoteCMD(s string) string {
	if s == "" {
		return `""`
	}
	if !needsQuote(s) {
		return s
	}
	escaped := strings.ReplaceAll(s, `"`, `\"`)
	return `"` + escaped + `"`
}

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '"', '\'', '\\', '$', '`', '|', '&', ';', '<', '>', '(', ')', '{', '}', '[', ']', '*', '?', '~', '#', '!':
			return true
		}
	}
	return false
}

// GlobalFlags returns docker CLI global flags for the connection (context or -H).
// Prefer --context when the Engine was resolved via Docker context; otherwise -H for
// explicit / DOCKER_HOST / remote endpoints. Local default sockets omit flags.
func GlobalFlags(conn ConnectionContext) []string {
	if conn.Source == docker.SourceContext {
		if c := strings.TrimSpace(conn.Context); c != "" {
			return []string{"--context", c}
		}
	}
	ep := strings.TrimSpace(conn.Endpoint)
	if ep == "" {
		return nil
	}
	switch conn.Source {
	case docker.SourceExplicit, docker.SourceDockerHost:
		return []string{"-H", ep}
	default:
		if isDefaultLocalEndpoint(ep) {
			return nil
		}
		if looksRemote(ep) {
			return []string{"-H", ep}
		}
		return nil
	}
}

func isDefaultLocalEndpoint(ep string) bool {
	switch ep {
	case "unix:///var/run/docker.sock", "npipe:////./pipe/docker_engine", "npipe:////./pipe/docker_engine_windows":
		return true
	default:
		return false
	}
}

func looksRemote(ep string) bool {
	lower := strings.ToLower(ep)
	return strings.HasPrefix(lower, "tcp://") ||
		strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "ssh://")
}

// RenderLine joins docker + global flags + args with shell-appropriate quoting.
func RenderLine(shell Shell, global []string, args []string) string {
	parts := make([]string, 0, 1+len(global)+len(args))
	parts = append(parts, "docker")
	for _, g := range global {
		parts = append(parts, Quote(shell, g))
	}
	for _, a := range args {
		parts = append(parts, Quote(shell, a))
	}
	return strings.Join(parts, " ")
}
