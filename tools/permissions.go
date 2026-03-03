package tools

import "encoding/json"

// HasPermission checks if a role's permissions JSONB grants the given resource and action.
// permissions can be:
//   - {"all": true} — grants all permissions (admin)
//   - {"resource": {"action": true}} — e.g. {"articles": {"create": true, "read": true}}
func HasPermission(permissions []byte, resource, action string) bool {
	if len(permissions) == 0 {
		return false
	}

	var perms map[string]interface{}
	if err := json.Unmarshal(permissions, &perms); err != nil {
		return false
	}

	// Admin: "all": true
	if allVal, exists := perms["all"]; exists {
		if allBool, ok := allVal.(bool); ok && allBool {
			return true
		}
	}

	// Specific resource.action
	if resourcePerms, exists := perms[resource]; exists {
		if resourceMap, ok := resourcePerms.(map[string]interface{}); ok {
			if actionVal, exists := resourceMap[action]; exists {
				if actionBool, ok := actionVal.(bool); ok && actionBool {
					return true
				}
			}
		}
	}

	return false
}
