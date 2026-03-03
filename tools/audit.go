// Package tools provides audit action and resource type constants for consistent audit logging.
package tools

// Audit actions
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionLogin  = "login"
	ActionLogout = "logout"
)

// Audit resource types (must match path/domain names used in API)
const (
	ResourceArticle   = "article"
	ResourceLot       = "lot"
	ResourceLocation  = "location"
	ResourceSerial    = "serial"
	ResourceInventory = "inventory"
)
