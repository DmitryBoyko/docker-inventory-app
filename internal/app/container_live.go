package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/redact"
	"github.com/epm-games/docker-visualizer/internal/store"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

const (
	defaultLogTail = 200
	maxLogTail     = 5000
	maxLogBytes    = 512 << 10 // 512 KiB
)

// ContainerLiveService performs on-demand Engine calls (inspect/logs).
type ContainerLiveService struct {
	Docker *docker.Client
	Store  *store.Store
}

// InspectResult is a (possibly redacted) inspect document.
type InspectResult struct {
	ID             string
	Name           string
	Redacted       bool
	RedactedFields []string
	Inspect        json.RawMessage
}

// Inspect fetches live inspect JSON. redact defaults to true when useDefault.
func (s *ContainerLiveService) Inspect(ctx context.Context, id string, doRedact bool) (InspectResult, error) {
	fullID, name, err := s.resolveID(id)
	if err != nil {
		return InspectResult{}, err
	}
	timeout := s.Docker.Timeout()
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := s.Docker.API().ContainerInspect(reqCtx, fullID, client.ContainerInspectOptions{})
	if err != nil {
		return InspectResult{}, fmt.Errorf("%s", docker.ClassifyError(s.Docker.Endpoint().Host, err))
	}
	raw := res.Raw
	if len(raw) == 0 {
		raw, err = json.Marshal(res.Container)
		if err != nil {
			return InspectResult{}, err
		}
	}
	out, fields, err := redact.InspectJSON(raw, doRedact)
	if err != nil {
		return InspectResult{}, err
	}
	return InspectResult{
		ID:             fullID,
		Name:           name,
		Redacted:       doRedact,
		RedactedFields: fields,
		Inspect:        out,
	}, nil
}

// LogsOptions controls a non-follow log snapshot.
type LogsOptions struct {
	Tail       int
	Since      string
	Timestamps bool
}

// LogsResult is a demultiplexed log snapshot (not persisted).
type LogsResult struct {
	ID         string
	Name       string
	Tail       int
	Since      string
	Timestamps bool
	Truncated  bool
	Text       string
}

// Logs fetches a bounded stdout+stderr snapshot (Follow=false).
func (s *ContainerLiveService) Logs(ctx context.Context, id string, opts LogsOptions) (LogsResult, error) {
	fullID, name, err := s.resolveID(id)
	if err != nil {
		return LogsResult{}, err
	}
	tail := opts.Tail
	if tail <= 0 {
		tail = defaultLogTail
	}
	if tail > maxLogTail {
		tail = maxLogTail
	}

	timeout := 15 * time.Second
	if t := s.Docker.Timeout(); t > timeout {
		timeout = t
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rc, err := s.Docker.API().ContainerLogs(reqCtx, fullID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Follow:     false,
		Tail:       strconv.Itoa(tail),
		Since:      opts.Since,
	})
	if err != nil {
		return LogsResult{}, fmt.Errorf("%s", docker.ClassifyError(s.Docker.Endpoint().Host, err))
	}
	defer rc.Close()

	limited := &io.LimitedReader{R: rc, N: int64(maxLogBytes + 1)}
	raw, err := io.ReadAll(limited)
	if err != nil {
		return LogsResult{}, err
	}
	truncated := len(raw) > maxLogBytes
	if truncated {
		raw = raw[:maxLogBytes]
	}

	text := demuxLogs(raw)
	return LogsResult{
		ID:         fullID,
		Name:       name,
		Tail:       tail,
		Since:      opts.Since,
		Timestamps: opts.Timestamps,
		Truncated:  truncated,
		Text:       text,
	}, nil
}

func demuxLogs(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var stdout, stderr bytes.Buffer
	_, err := stdcopy.StdCopy(&stdout, &stderr, bytes.NewReader(raw))
	if err != nil {
		// Likely TTY / non-multiplexed stream.
		return string(raw)
	}
	var b strings.Builder
	b.Write(stdout.Bytes())
	if stderr.Len() > 0 {
		if stdout.Len() > 0 && !strings.HasSuffix(stdout.String(), "\n") {
			b.WriteByte('\n')
		}
		b.Write(stderr.Bytes())
	}
	return b.String()
}

func (s *ContainerLiveService) resolveID(id string) (fullID, name string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", errNotFound
	}
	if s.Store != nil {
		if c, ok := s.Store.Load().GetContainer(id); ok {
			return c.ID, c.Name, nil
		}
	}
	// Allow direct Engine id when not in snapshot yet.
	if len(id) >= 12 {
		return id, id, nil
	}
	return "", "", errNotFound
}

var errNotFound = errors.New("container not found")

// IsNotFound reports inventory miss.
func IsNotFound(err error) bool {
	return errors.Is(err, errNotFound)
}
