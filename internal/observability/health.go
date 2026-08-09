package observability

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// CollectorHealth is a point-in-time view of one collector.
type CollectorHealth struct {
	Name           string    `json:"name"`
	OK             bool      `json:"ok"`
	LastSuccessAt  time.Time `json:"lastSuccessAt,omitempty"`
	LastErrorAt    time.Time `json:"lastErrorAt,omitempty"`
	LastError      string    `json:"lastError,omitempty"`
	LastDurationMs int64     `json:"lastDurationMs"`
	SuccessCount   uint64    `json:"successCount"`
	ErrorCount     uint64    `json:"errorCount"`
}

// Registry tracks collector success/error timings for diagnostics.
type Registry struct {
	mu sync.RWMutex
	by map[string]*collectorState
}

type collectorState struct {
	name           string
	mu             sync.Mutex
	lastSuccessAt  time.Time
	lastErrorAt    time.Time
	lastError      string
	lastDurationMs atomic.Int64
	successCount   atomic.Uint64
	errorCount     atomic.Uint64
}

// NewRegistry creates an empty health registry.
func NewRegistry() *Registry {
	return &Registry{by: map[string]*collectorState{}}
}

// RecordSuccess records a successful collect.
func (r *Registry) RecordSuccess(name string, d time.Duration) {
	st := r.get(name)
	st.mu.Lock()
	st.lastSuccessAt = time.Now().UTC()
	st.lastError = ""
	st.mu.Unlock()
	st.lastDurationMs.Store(d.Milliseconds())
	st.successCount.Add(1)
}

// RecordError records a failed collect.
func (r *Registry) RecordError(name string, d time.Duration, err error) {
	st := r.get(name)
	st.mu.Lock()
	st.lastErrorAt = time.Now().UTC()
	if err != nil {
		st.lastError = err.Error()
	} else {
		st.lastError = "error"
	}
	st.mu.Unlock()
	st.lastDurationMs.Store(d.Milliseconds())
	st.errorCount.Add(1)
}

func (r *Registry) get(name string) *collectorState {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.by[name]
	if !ok {
		st = &collectorState{name: name}
		r.by[name] = st
	}
	return st
}

// Snapshot returns all known collectors sorted by name.
func (r *Registry) Snapshot() []CollectorHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]CollectorHealth, 0, len(r.by))
	for _, st := range r.by {
		st.mu.Lock()
		ok := !st.lastSuccessAt.IsZero() && (st.lastErrorAt.IsZero() || !st.lastErrorAt.After(st.lastSuccessAt))
		h := CollectorHealth{
			Name:           st.name,
			OK:             ok,
			LastSuccessAt:  st.lastSuccessAt,
			LastErrorAt:    st.lastErrorAt,
			LastError:      st.lastError,
			LastDurationMs: st.lastDurationMs.Load(),
			SuccessCount:   st.successCount.Load(),
			ErrorCount:     st.errorCount.Load(),
		}
		st.mu.Unlock()
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
