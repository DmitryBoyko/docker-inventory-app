//go:build integration

package docker

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPing_Integration(t *testing.T) {
	if os.Getenv("DOCKER_VISUALIZER_INTEGRATION") == "" && os.Getenv("CI_DOCKER") == "" {
		// Still run when Docker is expected; skip only if explicitly disabled.
	}

	cli, err := Connect(Options{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cli.Close()

	st := cli.Ping(context.Background())
	if !st.Connected {
		t.Skipf("docker not available (host=%s source=%s): %s", st.Host, st.Source, st.Error)
	}
	if st.APIVersion == "" {
		t.Fatal("expected apiVersion from ping")
	}
	t.Logf("connected host=%s api=%s osType=%s source=%s context=%s",
		st.Host, st.APIVersion, st.OSType, st.Source, st.Context)
}
