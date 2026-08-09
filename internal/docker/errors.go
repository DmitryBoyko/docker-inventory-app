package docker

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"
	"syscall"
)

// ClassifyError maps low-level dial/API failures to actionable messages.
func ClassifyError(host string, err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "timeout"):
		return fmt.Sprintf(
			"Timed out connecting to %s. Check that the daemon is healthy and the host URL is correct.",
			host,
		)
	case errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist) ||
		strings.Contains(lower, "connect: no such file") ||
		strings.Contains(lower, "cannot find the file") ||
		strings.Contains(lower, "the system cannot find"):
		return fmt.Sprintf(
			"Docker endpoint not found at %s. Is Docker Engine / Docker Desktop running? "+
				"Set --docker-host or DOCKER_HOST, or select the correct Docker context.",
			host,
		)
	case errors.Is(err, os.ErrPermission) || strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "access is denied"):
		return fmt.Sprintf(
			"Permission denied connecting to %s. On Linux, add your user to the docker group "+
				"or run with an account that can access the Docker socket/pipe.",
			host,
		)
	case errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(lower, "connection refused"):
		return fmt.Sprintf(
			"Connection refused to %s. The Docker daemon does not appear to be listening.",
			host,
		)
	case strings.Contains(lower, "tls") || strings.Contains(lower, "x509") ||
		strings.Contains(lower, "certificate"):
		return fmt.Sprintf(
			"TLS error talking to %s. For tcp:// hosts set DOCKER_CERT_PATH / DOCKER_TLS_VERIFY as with the Docker CLI.",
			host,
		)
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		return ClassifyError(host, opErr.Err)
	}

	return fmt.Sprintf("Cannot connect to Docker at %s: %v", host, err)
}
