package app

import (
	"strings"

	"github.com/epm-games/docker-visualizer/internal/commands"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// CommandsService generates CLI commands from the Command Registry (no execution).
type CommandsService struct {
	Store  *store.Store
	Docker *docker.Client
	Host   string
}

// ConnectionContext builds CLI context from the current Engine endpoint.
func (s *CommandsService) ConnectionContext() commands.ConnectionContext {
	ctx := commands.ConnectionContext{HostName: s.Host}
	if s.Docker != nil {
		ep := s.Docker.Endpoint()
		ctx.Endpoint = ep.Host
		ctx.Source = ep.Source
		ctx.Context = ep.Context
	}
	return ctx
}

// ListDefinitions returns the static registry.
func (s *CommandsService) ListDefinitions() []commands.Definition {
	return commands.Registry()
}

// ForEntity renders commands for an entity kind + ref.
func (s *CommandsService) ForEntity(kind commands.EntityKind, ref string, shell commands.Shell) []commands.Rendered {
	ref = strings.TrimSpace(ref)
	if ref == "" && kind != commands.EntitySystem {
		return nil
	}
	// Prefer canonical names from inventory when id-like refs are given.
	ref = s.resolveRef(kind, ref)
	shells := []commands.Shell{commands.ShellBash, commands.ShellPowerShell, commands.ShellCMD}
	if shell != "" {
		shells = []commands.Shell{shell}
	}
	return commands.Generate(s.ConnectionContext(), commands.Target{Kind: kind, Ref: ref}, shells...)
}

func (s *CommandsService) resolveRef(kind commands.EntityKind, ref string) string {
	if s == nil || s.Store == nil || ref == "" {
		return ref
	}
	snap := s.Store.Load()
	switch kind {
	case commands.EntityContainer:
		if c, ok := snap.GetContainer(ref); ok && c != nil {
			if c.Name != "" {
				return c.Name
			}
			return c.ID
		}
	case commands.EntityNetwork:
		if n, ok := snap.GetNetwork(ref); ok && n != nil {
			if n.Name != "" {
				return n.Name
			}
			return n.ID
		}
	case commands.EntityVolume:
		if v, ok := snap.GetVolume(ref); ok && v != nil {
			return v.Name
		}
	case commands.EntityImage:
		if img, ok := snap.GetImage(ref); ok && img != nil {
			if len(img.RepoTags) > 0 {
				return img.RepoTags[0]
			}
			return img.ID
		}
	}
	return ref
}
