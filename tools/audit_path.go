package tools

import "strings"

// ResolveResourceTypeFromPath maps an API path to a stable audit resource type.
// Example:
//   /api/articles/...  -> ResourceArticle
//   /api/lots/...      -> ResourceLot
//   /api/locations/... -> ResourceLocation
//   /api/serials/...   -> ResourceSerial
//   /api/inventory/... -> ResourceInventory
//
// It is intentionally simple and based on path prefixes so it can be used
// from middleware or controllers without importing routing packages.
func ResolveResourceTypeFromPath(path string) string {
	path = strings.ToLower(path)

	switch {
	case strings.HasPrefix(path, "/api/articles"):
		return ResourceArticle
	case strings.HasPrefix(path, "/api/lots"):
		return ResourceLot
	case strings.HasPrefix(path, "/api/locations"):
		return ResourceLocation
	case strings.HasPrefix(path, "/api/serials"):
		return ResourceSerial
	case strings.HasPrefix(path, "/api/inventory"):
		return ResourceInventory
	default:
		return ""
	}
}

