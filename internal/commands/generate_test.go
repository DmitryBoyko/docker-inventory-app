package commands

import (
	"testing"

	"github.com/epm-games/docker-visualizer/internal/docker"
)

func TestGenerateContainerInspectAllShells(t *testing.T) {
	conn := ConnectionContext{Source: docker.SourceDefault, Endpoint: "unix:///var/run/docker.sock"}
	got := Generate(conn, Target{Kind: EntityContainer, Ref: "nginx-prod"}, ShellBash, ShellPowerShell, ShellCMD)
	var inspect []Rendered
	for _, r := range got {
		if r.DefinitionID == "container.inspect" {
			inspect = append(inspect, r)
		}
	}
	if len(inspect) != 3 {
		t.Fatalf("want 3 inspect renders, got %d", len(inspect))
	}
	want := map[Shell]string{
		ShellBash:       "docker inspect nginx-prod",
		ShellPowerShell: "docker inspect nginx-prod",
		ShellCMD:        "docker inspect nginx-prod",
	}
	for _, r := range inspect {
		if r.Command != want[r.Shell] {
			t.Errorf("%s: got %q want %q", r.Shell, r.Command, want[r.Shell])
		}
		if r.RiskLevel != RiskReadOnly {
			t.Errorf("risk=%s", r.RiskLevel)
		}
	}
}

func TestGenerateLogsStatsNetworkVolumeImageSystem(t *testing.T) {
	conn := ConnectionContext{}
	cases := []struct {
		id   string
		kind EntityKind
		ref  string
		sub  string
	}{
		{"container.logs", EntityContainer, "nginx", "logs"},
		{"container.stats", EntityContainer, "nginx", "stats"},
		{"network.inspect", EntityNetwork, "proxy", "network inspect"},
		{"volume.inspect", EntityVolume, "postgres-data", "volume inspect"},
		{"image.inspect", EntityImage, "alpine:3.19", "image inspect"},
		{"system.df", EntitySystem, "", "system df"},
	}
	for _, tc := range cases {
		def, ok := Lookup(tc.id)
		if !ok {
			t.Fatalf("missing %s", tc.id)
		}
		out := GenerateOne(conn, def, tc.ref, ShellBash)
		if len(out) != 1 {
			t.Fatalf("%s: len=%d", tc.id, len(out))
		}
		if !stringsContains(out[0].Command, tc.sub) {
			t.Errorf("%s command %q missing %q", tc.id, out[0].Command, tc.sub)
		}
	}
}

func TestContextAwareGeneration(t *testing.T) {
	conn := ConnectionContext{Source: docker.SourceContext, Context: "production", Endpoint: "ssh://prod"}
	out := GenerateOne(conn, mustDef(t, "container.inspect"), "api", ShellBash)
	if out[0].Command != "docker --context production inspect api" {
		t.Fatalf("got %q", out[0].Command)
	}

	connH := ConnectionContext{Source: docker.SourceExplicit, Endpoint: "tcp://192.168.1.10:2375"}
	outH := GenerateOne(connH, mustDef(t, "container.inspect"), "api", ShellBash)
	if outH[0].Command != "docker -H tcp://192.168.1.10:2375 inspect api" {
		t.Fatalf("got %q", outH[0].Command)
	}
}

func TestQuotingSpecialNames(t *testing.T) {
	conn := ConnectionContext{}
	name := "my container's app"
	out := GenerateOne(conn, mustDef(t, "container.inspect"), name, ShellBash, ShellPowerShell, ShellCMD)
	byShell := map[Shell]string{}
	for _, r := range out {
		byShell[r.Shell] = r.Command
	}
	if byShell[ShellBash] != `docker inspect 'my container'\''s app'` {
		t.Errorf("bash=%q", byShell[ShellBash])
	}
	if byShell[ShellPowerShell] != `docker inspect 'my container''s app'` {
		t.Errorf("ps=%q", byShell[ShellPowerShell])
	}
	if byShell[ShellCMD] != `docker inspect "my container's app"` {
		t.Errorf("cmd=%q", byShell[ShellCMD])
	}
}

func TestExecIsInteractive(t *testing.T) {
	def, _ := Lookup("container.exec")
	if def.RiskLevel != RiskInteractive || !def.RequiresTTY {
		t.Fatalf("exec risk=%s tty=%v", def.RiskLevel, def.RequiresTTY)
	}
}

func mustDef(t *testing.T, id string) Definition {
	t.Helper()
	d, ok := Lookup(id)
	if !ok {
		t.Fatalf("missing %s", id)
	}
	return d
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
