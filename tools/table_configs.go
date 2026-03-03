package tools

// ArticlesTableConfig returns the generic table configuration for articles.
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
		SearchFields: []string{"a.sku", "a.name"},
		DefaultWhere: "",
		SelectFields: "a.id, a.sku, a.name, a.presentation, a.is_active, a.created_at",
		CSVFields:    []string{"id", "sku", "name", "presentation", "is_active", "created_at"},
		CSVHeaders:   []string{"ID", "SKU", "Nombre", "Presentación", "Activo", "Creado en"},
		DefaultSortBy:  "created_at",
		DefaultSortDir: "desc",
	}
}

// LocationsTableConfig returns the generic table configuration for locations.
func LocationsTableConfig() TableConfig {
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
		SearchFields: []string{"l.location_code", "l.description", "l.zone"},
		DefaultWhere: "",
		SelectFields: "l.id, l.location_code, l.description, l.zone, l.type, l.is_active, l.created_at",
		CSVFields:    []string{"id", "location_code", "description", "zone", "type", "is_active", "created_at"},
		CSVHeaders:   []string{"ID", "Código", "Descripción", "Zona", "Tipo", "Activo", "Creado en"},
		DefaultSortBy:  "created_at",
		DefaultSortDir: "desc",
	}
}

// LotsTableConfig returns the generic table configuration for lots.
func LotsTableConfig() TableConfig {
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
		SearchFields: []string{"l.lot_number", "l.sku"},
		DefaultWhere: "",
		SelectFields: "l.id, l.lot_number, l.sku, l.quantity, l.expiration_date, l.status, l.created_at",
		CSVFields:    []string{"id", "lot_number", "sku", "quantity", "expiration_at", "status", "created_at"},
		CSVHeaders:   []string{"ID", "Lote", "SKU", "Cantidad", "Expira en", "Estado", "Creado en"},
		DefaultSortBy:  "created_at",
		DefaultSortDir: "desc",
	}
}

