package domain

import "testing"

func TestParseComposeLabels_Standalone(t *testing.T) {
	m := ParseComposeLabels(nil)
	if m.Project != StandaloneStack || m.Service != nil {
		t.Fatalf("%+v", m)
	}
	m = ParseComposeLabels(map[string]string{"foo": "bar"})
	if m.Project != StandaloneStack {
		t.Fatalf("%+v", m)
	}
}

func TestParseComposeLabels_ProjectService(t *testing.T) {
	m := ParseComposeLabels(map[string]string{
		LabelComposeProject:         "prod",
		LabelComposeService:         "web",
		LabelComposeContainerNumber: "2",
	})
	if m.Project != "prod" {
		t.Fatalf("project=%s", m.Project)
	}
	if m.Service == nil || *m.Service != "web" {
		t.Fatalf("service=%v", m.Service)
	}
	if m.ContainerNumber == nil || *m.ContainerNumber != 2 {
		t.Fatalf("num=%v", m.ContainerNumber)
	}
}
