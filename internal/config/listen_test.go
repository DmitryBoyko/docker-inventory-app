package config

import "testing"

func TestIsLoopbackListen(t *testing.T) {
	if !IsLoopbackListen("127.0.0.1:8080") {
		t.Fatal("expected loopback")
	}
	if !IsLoopbackListen("localhost:8080") {
		t.Fatal("expected localhost")
	}
	if IsLoopbackListen(":8080") {
		t.Fatal("bare port binds all interfaces")
	}
	if IsLoopbackListen("0.0.0.0:8080") {
		t.Fatal("0.0.0.0 is not loopback")
	}
}
