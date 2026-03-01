package routes

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterDocsRoutes registers dev-only endpoints: route list and OpenAPI spec generated from the router.
// Call after all other routes so the engine has the full list. Used by Swagger UI at /swagger/.
func RegisterDocsRoutes(r *gin.Engine) {
	api := r.Group("/api")
	docs := api.Group("/docs")
	docs.GET("/routes", func(c *gin.Context) {
		routes := r.Routes()
		list := make([]gin.H, 0, len(routes))
		for _, route := range routes {
			list = append(list, gin.H{
				"method": route.Method,
				"path":   route.Path,
			})
		}
		c.JSON(200, gin.H{"routes": list})
	})
	docs.GET("/openapi.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.JSON(200, buildOpenAPI(r))
	})
}

// pathToTag derives a Swagger tag from the path (e.g. /api/articles/ -> Articles, /health -> Health).
func pathToTag(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return "General"
	}
	parts := strings.Split(path, "/")
	if parts[0] == "api" && len(parts) > 1 {
		name := parts[1]
		if idx := strings.Index(name, "-"); idx > 0 {
			name = name[:idx] + " " + name[idx+1:]
		}
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return strings.ToUpper(parts[0][:1]) + parts[0][1:]
}

// openAPITags defines tag order and descriptions for Swagger UI grouping.
var openAPITags = []map[string]interface{}{
	{"name": "Health", "description": "Health and readiness"},
	{"name": "Authentication", "description": "Login and auth"},
	{"name": "Encryption", "description": "Encrypt / decrypt"},
	{"name": "Users", "description": "User management"},
	{"name": "Dashboard", "description": "Dashboard stats"},
	{"name": "Locations", "description": "Locations"},
	{"name": "Articles", "description": "Articles / products"},
	{"name": "Inventory", "description": "Inventory"},
	{"name": "Lots", "description": "Lots"},
	{"name": "Serials", "description": "Serials"},
	{"name": "Receiving tasks", "description": "Receiving tasks"},
	{"name": "Picking tasks", "description": "Picking tasks"},
	{"name": "Adjustments", "description": "Adjustments"},
	{"name": "Stock alerts", "description": "Stock alerts"},
	{"name": "Inventory movements", "description": "Inventory movements"},
	{"name": "Gamification", "description": "Gamification and badges"},
	{"name": "Presentations", "description": "Presentations"},
	{"name": "Docs", "description": "API docs (routes, OpenAPI spec)"},
	{"name": "General", "description": "Other"},
}

// buildOpenAPI returns an OpenAPI 3.0 spec with all paths from the engine (for Swagger UI / dev tracking).
func buildOpenAPI(r *gin.Engine) map[string]interface{} {
	paths := make(map[string]interface{})
	for _, route := range r.Routes() {
		if route.Path == "" {
			continue
		}
		pathItem, ok := paths[route.Path]
		if !ok {
			pathItem = make(map[string]interface{})
			paths[route.Path] = pathItem
		}
		method := strings.ToLower(route.Method)
		tag := pathToTag(route.Path)
		pathItem.(map[string]interface{})[method] = map[string]interface{}{
			"tags":    []string{tag},
			"summary": route.Path,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{"description": "Success"},
				"400": map[string]interface{}{"description": "Bad request"},
				"401": map[string]interface{}{"description": "Unauthorized"},
				"404": map[string]interface{}{"description": "Not found"},
			},
		}
	}
	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "eSTOCK API",
			"version": "1.0",
		},
		"tags": openAPITags,
		"paths": paths,
	}
}
