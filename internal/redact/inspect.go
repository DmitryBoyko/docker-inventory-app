package redact

import (
	"encoding/json"
	"strings"
)

const placeholder = "[redacted]"

// secretKeySubstrings match label / env key names (case-insensitive).
var secretKeySubstrings = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"access_key", "private_key", "privatekey", "credential", "auth",
	"bearer", "session", "cookie", "registryauth",
}

// InspectJSON redacts sensitive fields from a Docker inspect document.
// When enabled=false, returns raw unchanged.
func InspectJSON(raw json.RawMessage, enabled bool) (out json.RawMessage, fields []string, err error) {
	if !enabled || len(raw) == 0 {
		return raw, nil, nil
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, err
	}
	fields = redactValue("", &doc)
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, nil, err
	}
	return b, fields, nil
}

func redactValue(path string, v *any) []string {
	var fields []string
	switch x := (*v).(type) {
	case map[string]any:
		fields = append(fields, redactMap(path, x)...)
	case []any:
		for i := range x {
			p := path + "[]"
			fields = append(fields, redactValue(p, &x[i])...)
		}
	}
	return fields
}

func redactMap(path string, m map[string]any) []string {
	var fields []string
	for k, val := range m {
		p := k
		if path != "" {
			p = path + "." + k
		}
		lk := strings.ToLower(k)

		switch {
		case lk == "env":
			m[k] = []any{placeholder}
			fields = append(fields, p)
			continue
		case lk == "registryauth" || lk == "password" || lk == "identitytoken":
			m[k] = placeholder
			fields = append(fields, p)
			continue
		case looksSecretKey(lk):
			switch val.(type) {
			case string, float64, bool, nil:
				m[k] = placeholder
				fields = append(fields, p)
				continue
			}
		}

		vv := val
		fields = append(fields, redactValue(p, &vv)...)
		m[k] = vv
	}
	return fields
}

func looksSecretKey(key string) bool {
	for _, s := range secretKeySubstrings {
		if strings.Contains(key, s) {
			return true
		}
	}
	return false
}

// EnvLine redacts the value part of KEY=value (for log filters / future use).
func EnvLine(line string) string {
	i := strings.IndexByte(line, '=')
	if i <= 0 {
		return line
	}
	key := line[:i]
	if looksSecretKey(strings.ToLower(key)) {
		return key + "=" + placeholder
	}
	return line
}
