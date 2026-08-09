package redact

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInspectJSONRedactsEnvAndSecrets(t *testing.T) {
	raw := json.RawMessage(`{
		"Id": "x",
		"Config": {
			"Env": ["FOO=bar", "PASSWORD=secret"],
			"Labels": {"com.app.name": "web", "api_token": "abc"}
		},
		"HostConfig": {
			"RegistryAuth": "supersecret"
		}
	}`)
	out, fields, err := InspectJSON(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) < 3 {
		t.Fatalf("fields=%v", fields)
	}
	s := string(out)
	if strings.Contains(s, "supersecret") || strings.Contains(s, "PASSWORD=secret") || strings.Contains(s, `"abc"`) {
		t.Fatalf("leak in %s", s)
	}
	if !strings.Contains(s, placeholder) {
		t.Fatalf("expected placeholder in %s", s)
	}

	passthrough, f2, err := InspectJSON(raw, false)
	if err != nil || len(f2) != 0 || string(passthrough) != string(raw) {
		t.Fatalf("passthrough failed: %v %v %s", err, f2, passthrough)
	}
}

func TestEnvLine(t *testing.T) {
	if EnvLine("PASSWORD=x") != "PASSWORD="+placeholder {
		t.Fatal(EnvLine("PASSWORD=x"))
	}
	if EnvLine("FOO=bar") != "FOO=bar" {
		t.Fatal(EnvLine("FOO=bar"))
	}
}
