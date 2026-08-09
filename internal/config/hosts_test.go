package config

import "testing"

func TestParseDockerHosts_Default(t *testing.T) {
	h, err := ParseDockerHosts("", "unix:///var/run/docker.sock")
	if err != nil || len(h) != 1 || h[0].Name != DefaultHostName || h[0].URL != "unix:///var/run/docker.sock" {
		t.Fatalf("%+v err=%v", h, err)
	}
}

func TestParseDockerHosts_List(t *testing.T) {
	h, err := ParseDockerHosts("local=npipe:////./pipe/docker_engine,lab=tcp://192.168.1.10:2376", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 2 || h[0].Name != "local" || h[1].URL != "tcp://192.168.1.10:2376" {
		t.Fatalf("%+v", h)
	}
}

func TestParseDockerHosts_Bad(t *testing.T) {
	if _, err := ParseDockerHosts("nouserurl", ""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseDockerHosts("a=u,a=v", ""); err == nil {
		t.Fatal("duplicate")
	}
}
