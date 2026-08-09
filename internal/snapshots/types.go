// Package snapshots persists sanitized inventory snapshots and computes diffs.
package snapshots

import (
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

// Meta is list/header information without full payload.
type Meta struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	HostName      string    `json:"hostName"`
	Endpoint      string    `json:"endpoint,omitempty"`
	Context       string    `json:"context,omitempty"`
	DockerVersion string    `json:"dockerVersion,omitempty"`
	Label         string    `json:"label,omitempty"`
	Counts        Counts    `json:"counts"`
}

// Counts summarizes inventory sizes.
type Counts struct {
	Containers int `json:"containers"`
	Images     int `json:"images"`
	Networks   int `json:"networks"`
	Volumes    int `json:"volumes"`
	Stacks     int `json:"stacks"`
}

// Snapshot is a persisted, sanitized inventory point-in-time view.
// Environment variables and secrets are never stored.
type Snapshot struct {
	Meta
	Containers []ContainerView `json:"containers"`
	Images     []ImageView     `json:"images"`
	Networks   []NetworkView   `json:"networks"`
	Volumes    []VolumeView    `json:"volumes"`
	Stacks     []StackView     `json:"stacks"`
}

// ContainerView is a redacted container row for snapshots.
type ContainerView struct {
	ID           string               `json:"id"`
	IDShort      string               `json:"idShort"`
	Name         string               `json:"name"`
	Stack        string               `json:"stack,omitempty"`
	Service      string               `json:"service,omitempty"`
	Image        string               `json:"image"`
	State        domain.ContainerState `json:"state"`
	Health       domain.HealthState   `json:"health"`
	RestartCount int                  `json:"restartCount"`
	WritableBytes *int64              `json:"writableBytes,omitempty"`
	MemoryBytes  *int64               `json:"memoryBytes,omitempty"`
	CPUPercent   *float64             `json:"cpuPercent,omitempty"`
	// Labels are filtered — only compose-related keys kept.
	Labels map[string]string `json:"labels,omitempty"`
}

// ImageView is a sanitized image row.
type ImageView struct {
	ID             string   `json:"id"`
	IDShort        string   `json:"idShort"`
	RepoTags       []string `json:"repoTags,omitempty"`
	SizeBytes      int64    `json:"sizeBytes"`
	ContainerCount int      `json:"containerCount"`
	Dangling       bool     `json:"dangling"`
}

// NetworkView is a sanitized network row.
type NetworkView struct {
	ID         string   `json:"id"`
	IDShort    string   `json:"idShort"`
	Name       string   `json:"name"`
	Driver     string   `json:"driver"`
	Internal   bool     `json:"internal"`
	Containers []string `json:"containers,omitempty"`
}

// VolumeView is a sanitized volume row (no mountpoint secrets beyond path).
type VolumeView struct {
	Name           string  `json:"name"`
	Driver         string  `json:"driver"`
	UsageBytes     *int64  `json:"usageBytes,omitempty"`
	UsageAvailable bool    `json:"usageAvailable"`
	Containers     []string `json:"containers,omitempty"`
	Stacks         []string `json:"stacks,omitempty"`
}

// StackView is a lightweight stack rollup.
type StackView struct {
	Name           string `json:"name"`
	ContainerCount int    `json:"containerCount"`
	RunningCount   int    `json:"runningCount"`
}

// ChangeKind for diff entries.
type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeRemoved  ChangeKind = "removed"
	ChangeModified ChangeKind = "modified"
)

// Diff is a comparison result.
type Diff struct {
	LeftID    string      `json:"leftId"`
	RightID   string      `json:"rightId"`
	LeftAt    time.Time   `json:"leftAt,omitempty"`
	RightAt   time.Time   `json:"rightAt,omitempty"`
	Containers []Change   `json:"containers"`
	Images     []Change   `json:"images"`
	Networks   []Change   `json:"networks"`
	Volumes    []Change   `json:"volumes"`
	Stacks     []Change   `json:"stacks"`
}

// Change describes one entity delta.
type Change struct {
	Kind    ChangeKind     `json:"kind"`
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Fields  []FieldChange  `json:"fields,omitempty"`
}

// FieldChange is a single field delta.
type FieldChange struct {
	Field string `json:"field"`
	From  any    `json:"from,omitempty"`
	To    any    `json:"to,omitempty"`
}
