package web

import "strings"

func sanitizeAlertKeyPart(label string) string {
	slug := strings.ToLower(strings.TrimSpace(label))
	slug = strings.NewReplacer(" ", "_", "/", "_", ":", "_", "-", "_").Replace(slug)
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	return strings.Trim(slug, "_")
}
