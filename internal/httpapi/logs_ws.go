package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/epm-games/docker-visualizer/internal/app"
)

// handleContainerLogsWS streams follow logs for one container (V2).
func (s *Server) handleContainerLogsWS(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	id := r.PathValue("id")
	q := r.URL.Query()
	tail := 0
	if v := q.Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			tail = n
		}
	}
	opts := app.StreamLogsOptions{
		Tail:       tail,
		Since:      q.Get("since"),
		Timestamps: parseBoolDefaultFalse(q.Get("timestamps")),
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(1 << 20)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		defer cancel()
		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				return
			}
		}
	}()

	_ = writeLogsWS(ctx, conn, map[string]any{
		"type":      "container.logs.status",
		"host":      rt.Name,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"id":        id,
			"connected": true,
			"follow":    true,
		},
	})

	err = rt.Live.StreamLogs(ctx, id, opts, func(chunk string) error {
		if chunk == "" {
			return nil
		}
		return writeLogsWS(ctx, conn, map[string]any{
			"type":      "container.logs",
			"host":      rt.Name,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"data": map[string]any{
				"id":   id,
				"text": chunk,
			},
		})
	})
	if err != nil && ctx.Err() == nil {
		code := "docker_error"
		msg := err.Error()
		if app.IsNotFound(err) {
			code = "not_found"
			msg = "container not found"
		}
		_ = writeLogsWS(ctx, conn, map[string]any{
			"type":      "error",
			"host":      rt.Name,
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"data":      map[string]string{"code": code, "message": msg},
		})
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

func writeLogsWS(ctx context.Context, conn *websocket.Conn, v any) error {
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(wctx, websocket.MessageText, b)
}
