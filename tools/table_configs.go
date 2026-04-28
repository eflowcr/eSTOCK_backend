package tools

// ArticlesTableConfig returns the generic table configuration for articles.
//
// S3.5 W1 — articles is now tenant-scoped (HR-S3-W5 C2). Use ArticlesTableConfigForTenant
// from HTTP routes; this no-arg variant is kept for back-compat / single-tenant callers
// and emits a wide-open SELECT (no tenant filter). It MUST NOT be used by tenant-aware
// HTTP handlers — they will leak data across tenants.
func ArticlesTableConfig() TableConfig {
	return TableConfig{
		EntityName: "artículos",
		FromClause: "articles a",
		AllowedCols: map[string]string{
			"id":           "a.id",
			"sku":          "a.sku",
			"name":         "a.name",
			"presentation": "a.presentation",
			"is_active":    "a.is_active",
			"created_at":   "a.created_at",
		},
		SearchFields:   []string{"a.sku", "a.name"},
		DefaultWhere:   "",
		SelectFields:   "a.id, a.sku, a.name, a.presentation, a.is_active, a.created_at",
		CSVFields:      []string{"id", "sku", "name", "presentation", "is_active", "created_at"},
		CSVHeaders:     []string{"ID", "SKU", "Nombre", "Presentación", "Activo", "Creado en"},
		DefaultSortBy:  "created_at",
		DefaultSortDir: "desc",
	}
}

// ArticlesTableConfigForTenant returns ArticlesTableConfig with a tenant-scoped
// DefaultWhere clause baked in. tenantID must be a UUID — the value is sanitised
// (only [0-9a-f-] kept) before being inlined into the SQL fragment, so it is safe
// to interpolate without a placeholder. The generic table handler doesn't expose
// argument injection for DefaultWhere, hence the inline approach.
func ArticlesTableConfigForTenant(tenantID string) TableConfig {
	cfg := ArticlesTableConfig()
	cfg.DefaultWhere = "a.tenant_id = '" + sanitizeUUID(tenantID) + "'::uuid"
	return cfg
}

// sanitizeUUID drops any character that isn't valid in a hex/dash UUID. Defensive.
func sanitizeUUID(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-' {
			out = append(out, c)
		}
	}
	return string(out)
}

// LocationsTableConfig returns the generic table configuration for locations.
//
// S3.5 W2-A: tenantID is baked into DefaultWhere so the generic /table and
// /table/export endpoints filter by tenant. The literal cast is safe because
// tenantID comes from server-side configuration (configuration.Config.TenantID),
// never from user input.
func LocationsTableConfig(tenantID string) TableConfig {
	return TableConfig{
		EntityName: "ubicaciones",
		FromClause: "locations l",
		AllowedCols: map[string]string{
			"id":            "l.id",
			"location_code": "l.location_code",
			"description":   "l.description",
			"zone":          "l.zone",
			"type":          "l.type",
			"is_active":     "l.is_active",
			"created_at":    "l.created_at",
		},
		SearchFields:   []string{"l.location_code", "l.description", "l.zone"},
		DefaultWhere:   "l.tenant_id = '" + tenantID + "'::uuid",
		SelectFields:   "l.id, l.location_code, l.description, l.zone, l.type, l.is_active, l.created_at",
		CSVFields:      []string{"id", "location_code", "description", "zone", "type", "is_active", "created_at"},
		CSVHeaders:     []string{"ID", "Código", "Descripción", "Zona", "Tipo", "Activo", "Creado en"},
		DefaultSortBy:  "created_at",
		DefaultSortDir: "desc",
	}
}

// LotsTableConfig returns the generic table configuration for lots.
//
// S3.5 W2-B: tenantID, when non-empty, is baked into DefaultWhere as a UUID literal so
// the generic list/export handlers always scope rows to the calling tenant. The literal
// is single-quoted and ::uuid-cast — Postgres rejects malformed UUIDs at plan time, so
// an invalid string from a misconfigured route surfaces as a SQL error rather than a
// silent cross-tenant leak.
func LotsTableConfig(tenantID ...string) TableConfig {
	tid := ""
	if len(tenantID) > 0 {
		tid = tenantID[0]
	}
	defaultWhere := ""
	if tid != "" {
		defaultWhere = "l.tenant_id = '" + tid + "'::uuid"
	}
	return TableConfig{
		EntityName: "lotes",
		FromClause: "lots l",
		AllowedCols: map[string]string{
			"id":            "l.id",
			"lot_number":    "l.lot_number",
			"sku":           "l.sku",
			"quantity":      "l.quantity",
			"expiration_at": "l.expiration_date",
			"status":        "l.status",
			"created_at":    "l.created_at",
		},
		SearchFields:   []string{"l.lot_number", "l.sku"},
		DefaultWhere:   defaultWhere,
		SelectFields:   "l.id, l.lot_number, l.sku, l.quantity, l.expiration_date, l.status, l.created_at",
		CSVFields:      []string{"id", "lot_number", "sku", "quantity", "expiration_at", "status", "created_at"},
		CSVHeaders:     []string{"ID", "Lote", "SKU", "Cantidad", "Expira en", "Estado", "Creado en"},
		DefaultSortBy:  "created_at",
		DefaultSortDir: "desc",
	}
}

