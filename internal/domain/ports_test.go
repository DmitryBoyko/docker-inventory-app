package domain

import "testing"

func TestClassifyHostIP(t *testing.T) {
	cases := map[string]PortExposure{
		"0.0.0.0":      PortExposurePublic,
		"":             PortExposurePublic,
		"*":            PortExposurePublic,
		"::":           PortExposurePublic,
		"[::]":         PortExposurePublic,
		"127.0.0.1":    PortExposureLocalhost,
		"::1":          PortExposureLocalhost,
		"[::1]":        PortExposureLocalhost,
		"192.168.1.10": PortExposureSpecific,
	}
	for in, want := range cases {
		if got := ClassifyHostIP(in); got != want {
			t.Fatalf("%q: got %s want %s", in, got, want)
		}
	}
}

func TestFormatPortsPSStyle_Public(t *testing.T) {
	hp := uint16(443)
	ports := []Port{
		{HostIP: "0.0.0.0", HostPort: &hp, ContainerPort: 443, Protocol: "tcp", Exposure: PortExposurePublic},
		{HostIP: "::", HostPort: &hp, ContainerPort: 443, Protocol: "tcp", Exposure: PortExposurePublic},
	}
	ext, intn := FormatPortsPSStyle(ports)
	if ext != "*:443->443/tcp [наружу]" {
		t.Fatalf("external=%q", ext)
	}
	if intn != "-" {
		t.Fatalf("internal=%q", intn)
	}
}

func TestFormatPortsPSStyle_LocalhostAndInternal(t *testing.T) {
	hp := uint16(8080)
	ports := []Port{
		{HostIP: "127.0.0.1", HostPort: &hp, ContainerPort: 80, Protocol: "tcp", Exposure: PortExposureLocalhost},
		{ContainerPort: 80, Protocol: "tcp", Exposure: PortExposureInternal},
	}
	ext, intn := FormatPortsPSStyle(ports)
	if ext != "127.0.0.1:8080->80/tcp [localhost]" {
		t.Fatalf("external=%q", ext)
	}
	if intn != "80/tcp" {
		t.Fatalf("internal=%q", intn)
	}
}

func TestBuildExternalExposureAndSummary(t *testing.T) {
	hp80 := uint16(80)
	hp8080 := uint16(8080)
	ports := []Port{
		{HostIP: "0.0.0.0", HostPort: &hp80, ContainerPort: 80, Protocol: "tcp", Exposure: PortExposurePublic},
		{HostIP: "127.0.0.1", HostPort: &hp8080, ContainerPort: 8080, Protocol: "tcp", Exposure: PortExposureLocalhost},
		{ContainerPort: 5432, Protocol: "tcp", Exposure: PortExposureInternal},
	}
	routes := BuildExternalExposure(ports)
	if len(routes) != 2 {
		t.Fatalf("routes=%d %+v", len(routes), routes)
	}
	if routes[0].HostIP != "*" || routes[0].Scope != ExposureScopeExternal {
		t.Fatalf("route0=%+v", routes[0])
	}
	if routes[1].Scope != ExposureScopeLocalhost {
		t.Fatalf("route1=%+v", routes[1])
	}
	if SummarizeExposure(ports) != ExposureScopeExternal {
		t.Fatalf("summary=%s", SummarizeExposure(ports))
	}
	if SummarizeExposure(nil) != ExposureScopeInternal {
		t.Fatalf("empty summary")
	}
}

func TestMapPortBindings(t *testing.T) {
	ports := MapPortBindings([]PortBindingInput{
		{HostIP: "0.0.0.0", HostPort: 80, ContainerPort: 80, Protocol: "tcp", Published: true},
		{ContainerPort: 53, Protocol: "udp", Published: false},
	})
	if len(ports) != 2 {
		t.Fatalf("len=%d", len(ports))
	}
	var pub, intern int
	for _, p := range ports {
		switch p.Exposure {
		case PortExposurePublic:
			pub++
		case PortExposureInternal:
			intern++
		}
	}
	if pub != 1 || intern != 1 {
		t.Fatalf("pub=%d internal=%d ports=%+v", pub, intern, ports)
	}
}

