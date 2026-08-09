package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
)

func (s *Server) handleMetricsHistory(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Metrics == nil || rt.Metrics.DB == nil {
		writeErr(w, http.StatusServiceUnavailable, "metrics_disabled", "historical metrics are disabled")
		return
	}

	q := r.URL.Query()
	from, err := parseTimeQuery(q.Get("from"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid from: "+err.Error())
		return
	}
	to, err := parseTimeQuery(q.Get("to"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid to: "+err.Error())
		return
	}
	step, err := parseDurationQuery(q.Get("step"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid step: "+err.Error())
		return
	}

	res, err := rt.Metrics.History(app.MetricsHistoryQuery{
		Scope: q.Get("scope"),
		ID:    firstNonEmpty(q.Get("id"), q.Get("containerId")),
		From:  from,
		To:    to,
		Step:  step,
	})
	if err != nil {
		msg := err.Error()
		switch {
		case strings.HasPrefix(msg, "metrics_disabled:"):
			writeErr(w, http.StatusServiceUnavailable, "metrics_disabled", strings.TrimSpace(strings.TrimPrefix(msg, "metrics_disabled:")))
		case strings.HasPrefix(msg, "bad_request:"):
			writeErr(w, http.StatusBadRequest, "bad_request", strings.TrimSpace(strings.TrimPrefix(msg, "bad_request:")))
		default:
			writeErr(w, http.StatusInternalServerError, "metrics_error", msg)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      res.Host,
		"data":      res,
	})
}

func parseTimeQuery(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Time{}, errString("want RFC3339")
}

type errString string

func (e errString) Error() string { return string(e) }

func parseDurationQuery(v string) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
	return time.ParseDuration(v)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
