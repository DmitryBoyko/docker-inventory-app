package app

import "github.com/epm-games/docker-visualizer/internal/uiembed"

// SettingsView is the read-only process configuration for the Settings UI.
type SettingsView struct {
	Listen         string            `json:"listen"`
	ListenLoopback bool              `json:"listenLoopback"`
	AuthEnabled    bool              `json:"authEnabled"`
	DockerTimeout  string            `json:"dockerTimeout"`
	Intervals      map[string]string `json:"intervals"`
	Version        string            `json:"version"`
	Commit         string            `json:"commit"`
	UIEmbedded     bool              `json:"uiEmbedded"`
	Defaults       SettingsDefaults  `json:"defaults"`
}

// SettingsDefaults are server-side defaults the UI should follow.
type SettingsDefaults struct {
	InspectRedact bool `json:"inspectRedact"`
}

// Settings returns the public settings payload (never includes the auth token).
func (s *DiagnosticsService) Settings() SettingsView {
	intervals := s.Intervals
	if intervals == nil {
		intervals = map[string]string{}
	}
	return SettingsView{
		Listen:         s.Listen,
		ListenLoopback: isListenLoopback(s.Listen),
		AuthEnabled:    s.AuthEnabled,
		DockerTimeout:  s.DockerTimeout,
		Intervals:      intervals,
		Version:        s.Version,
		Commit:         s.Commit,
		UIEmbedded:     uiembed.Available(),
		Defaults: SettingsDefaults{
			InspectRedact: true,
		},
	}
}
