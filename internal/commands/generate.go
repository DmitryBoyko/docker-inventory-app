package commands

import "strings"

// Generate renders all applicable commands for a target across supported shells.
func Generate(conn ConnectionContext, target Target, shells ...Shell) []Rendered {
	if len(shells) == 0 {
		shells = []Shell{ShellBash, ShellPowerShell, ShellCMD}
	}
	ref := strings.TrimSpace(target.Ref)
	if ref == "" {
		ref = strings.TrimSpace(target.ID)
	}
	defs := ForEntity(target.Kind)
	global := GlobalFlags(conn)
	out := make([]Rendered, 0, len(defs)*len(shells))
	for _, d := range defs {
		args := expandArgs(d.ArgsTemplate, ref)
		for _, sh := range shells {
			if !supports(d, sh) {
				continue
			}
			out = append(out, Rendered{
				DefinitionID:   d.ID,
				TitleKey:       d.TitleKey,
				DescriptionKey: d.DescriptionKey,
				Title:          d.Title,
				Description:    d.Description,
				Category:       d.Category,
				EntityKind:     d.EntityKind,
				RiskLevel:      d.RiskLevel,
				RequiresTTY:    d.RequiresTTY,
				Shell:          sh,
				Command:        RenderLine(sh, global, args),
				EntityRef:      ref,
			})
		}
	}
	return out
}

// GenerateOne renders a single definition for the given shells.
func GenerateOne(conn ConnectionContext, def Definition, ref string, shells ...Shell) []Rendered {
	if len(shells) == 0 {
		shells = []Shell{ShellBash, ShellPowerShell, ShellCMD}
	}
	ref = strings.TrimSpace(ref)
	args := expandArgs(def.ArgsTemplate, ref)
	global := GlobalFlags(conn)
	out := make([]Rendered, 0, len(shells))
	for _, sh := range shells {
		if !supports(def, sh) {
			continue
		}
		out = append(out, Rendered{
			DefinitionID:   def.ID,
			TitleKey:       def.TitleKey,
			DescriptionKey: def.DescriptionKey,
			Title:          def.Title,
			Description:    def.Description,
			Category:       def.Category,
			EntityKind:     def.EntityKind,
			RiskLevel:      def.RiskLevel,
			RequiresTTY:    def.RequiresTTY,
			Shell:          sh,
			Command:        RenderLine(sh, global, args),
			EntityRef:      ref,
		})
	}
	return out
}

func expandArgs(tmpl []string, ref string) []string {
	out := make([]string, len(tmpl))
	for i, t := range tmpl {
		out[i] = strings.ReplaceAll(t, "{{ref}}", ref)
	}
	return out
}

func supports(d Definition, sh Shell) bool {
	switch sh {
	case ShellBash:
		return d.SupportsBash
	case ShellPowerShell:
		return d.SupportsPowerShell
	case ShellCMD:
		return d.SupportsCMD
	default:
		return false
	}
}
