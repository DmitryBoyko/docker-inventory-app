package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/metricsdb"
)

// MetricsService queries historical samples (ADR-015).
type MetricsService struct {
	DB   *metricsdb.Store
	Host string // default host name for this runtime
}

// MetricsHistoryQuery selects a series window.
type MetricsHistoryQuery struct {
	Scope string // host | container
	ID    string // container id when scope=container
	From  time.Time
	To    time.Time
	Step  time.Duration
}

// MetricsHistoryResult is the API payload.
type MetricsHistoryResult struct {
	Scope       string            `json:"scope"`
	Host        string            `json:"host"`
	ID          string            `json:"id,omitempty"`
	From        time.Time         `json:"from"`
	To          time.Time         `json:"to"`
	StepSeconds int64             `json:"stepSeconds"`
	Points      []metricsdb.Point `json:"points"`
}

// History returns a time series or an error with a stable code prefix for HTTP mapping.
func (s *MetricsService) History(q MetricsHistoryQuery) (MetricsHistoryResult, error) {
	if s == nil || s.DB == nil {
		return MetricsHistoryResult{}, fmt.Errorf("metrics_disabled: historical metrics are off")
	}
	scope := strings.ToLower(strings.TrimSpace(q.Scope))
	if scope == "" {
		scope = "host"
	}
	host := s.Host
	if host == "" {
		host = "default"
	}
	from, to := q.From, q.To
	if to.IsZero() {
		to = time.Now().UTC()
	}
	if from.IsZero() {
		from = to.Add(-time.Hour)
	}
	step := q.Step
	if step <= 0 {
		step = s.DB.SampleInterval()
	}

	var (
		pts []metricsdb.Point
		err error
		id  string
	)
	switch scope {
	case "host":
		pts, err = s.DB.QueryHost(host, from, to, step)
	case "container":
		id = strings.TrimSpace(q.ID)
		if id == "" {
			return MetricsHistoryResult{}, fmt.Errorf("bad_request: id required for scope=container")
		}
		pts, err = s.DB.QueryContainer(host, id, from, to, step)
	default:
		return MetricsHistoryResult{}, fmt.Errorf("bad_request: scope must be host or container")
	}
	if err != nil {
		return MetricsHistoryResult{}, err
	}
	if pts == nil {
		pts = []metricsdb.Point{}
	}
	return MetricsHistoryResult{
		Scope:       scope,
		Host:        host,
		ID:          id,
		From:        from.UTC(),
		To:          to.UTC(),
		StepSeconds: int64(step / time.Second),
		Points:      pts,
	}, nil
}
