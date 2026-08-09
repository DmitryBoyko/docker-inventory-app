package domain

import "strings"

// ShortID returns the 12-char Docker short ID.
func ShortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

// ContainerName strips the leading slash from Docker Names.
func ContainerName(names []string, fallback string) string {
	if len(names) == 0 {
		return strings.TrimPrefix(fallback, "/")
	}
	n := names[0]
	return strings.TrimPrefix(n, "/")
}

// ShortImage matches PowerShell Get-ShortImage for sha256 refs.
func ShortImage(image string) string {
	if image == "" {
		return "-"
	}
	if strings.HasPrefix(image, "sha256:") && len(image) >= 19 {
		return "sha256:" + image[7:19] + "..."
	}
	return image
}
