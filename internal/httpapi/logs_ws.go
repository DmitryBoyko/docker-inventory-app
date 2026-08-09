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
// Query: tail, timestamps, since. Auth via same /api middleware as REST.
func (s *Server) handleContainerLogsWS(w http.ResponseWriter, r *http.Request) {
	if s.Live == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "live container API not initialized")
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

	// Client close / ping reader.
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
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"id":        id,
			"connected": true,
			"follow":    true,
		},
	})

	err = s.Live.StreamLogs(ctx, id, opts, func(chunk string) error {
		if chunk == "" {
			return nil
		}
		return writeLogsWS(ctx, conn, map[string]any{
			"type":      "container.logs",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"data": map[string]any{
				"id":   id,
				"text": chunk,
			},
		})
	})
	if err != nil && ctx.Err() == nil {
		if app.IsNotFound(err) {
			_ = writeLogsWS(ctx, conn, map[string]any{
				"type":      "error",
				"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
				"data":      map[string]string{"code": "not_found", "message": "container not found"},
			})
		} else {
			_ = writeLogsWS(ctx, conn, map[string]any{
				"type":      "error",
				"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
				"data":      map[string]string{"code": "docker_error", "message": err.Error()},
			})
		}
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
