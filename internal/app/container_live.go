package app

import (
	"bufio"
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

// StreamLogsOptions controls a follow log stream.
type StreamLogsOptions struct {
	Tail       int
	Since      string
	Timestamps bool
}

// StreamLogs follows container logs until ctx is cancelled. Emits demuxed text chunks
// (often whole lines). Does not persist logs.
func (s *ContainerLiveService) StreamLogs(ctx context.Context, id string, opts StreamLogsOptions, emit func(chunk string) error) error {
	fullID, _, err := s.resolveID(id)
	if err != nil {
		return err
	}
	tail := opts.Tail
	if tail <= 0 {
		tail = defaultLogTail
	}
	if tail > maxLogTail {
		tail = maxLogTail
	}
	if emit == nil {
		return errors.New("emit required")
	}

	rc, err := s.Docker.API().ContainerLogs(ctx, fullID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Follow:     true,
		Tail:       strconv.Itoa(tail),
		Since:      opts.Since,
	})
	if err != nil {
		return fmt.Errorf("%s", docker.ClassifyError(s.Docker.Endpoint().Host, err))
	}
	defer rc.Close()

	br := bufio.NewReader(rc)
	peek, _ := br.Peek(8)
	if looksLikeStdcopy(peek) {
		stdout := &chunkWriter{emit: emit}
		stderr := &chunkWriter{emit: emit}
		_, copyErr := stdcopy.StdCopy(stdout, stderr, br)
		_ = stdout.flush()
		_ = stderr.flush()
		if ctx.Err() != nil || errors.Is(copyErr, context.Canceled) || errors.Is(copyErr, io.EOF) || copyErr == nil {
			return nil
		}
		return copyErr
	}
	return streamRawLines(ctx, br, emit)
}

func looksLikeStdcopy(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	// stdcopy header: stream (0-2) + 3 zero bytes + big-endian size.
	return b[0] <= 2 && b[1] == 0 && b[2] == 0 && b[3] == 0
}

type chunkWriter struct {
	buf  []byte
	emit func(string) error
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			// Cap unbounded mid-line buffer.
			if len(w.buf) > 64<<10 {
				chunk := string(w.buf)
				w.buf = w.buf[:0]
				if err := w.emit(chunk); err != nil {
					return 0, err
				}
			}
			return len(p), nil
		}
		line := string(w.buf[:i+1])
		w.buf = w.buf[i+1:]
		if err := w.emit(line); err != nil {
			return 0, err
		}
	}
}

func (w *chunkWriter) flush() error {
	if len(w.buf) == 0 {
		return nil
	}
	chunk := string(w.buf)
	w.buf = w.buf[:0]
	return w.emit(chunk)
}

func streamRawLines(ctx context.Context, r io.Reader, emit func(string) error) error {
	buf := make([]byte, 32<<10)
	var pending []byte
	for {
		if ctx.Err() != nil {
			return nil
		}
		n, err := r.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			for {
				i := bytes.IndexByte(pending, '\n')
				if i < 0 {
					break
				}
				if e := emit(string(pending[:i+1])); e != nil {
					return e
				}
				pending = pending[i+1:]
			}
			if len(pending) > 64<<10 {
				if e := emit(string(pending)); e != nil {
					return e
				}
				pending = pending[:0]
			}
		}
		if err != nil {
			if len(pending) > 0 {
				_ = emit(string(pending))
			}
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}
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
