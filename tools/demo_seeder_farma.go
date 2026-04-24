package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const FarmaSeedName = "farma-50skus"

// tenantPrefix builds a short, deterministic prefix from a tenant UUID so demo
// data inserted for two different tenants does not collide on still-global
// UNIQUE constraints (e.g. articles_sku_key, receiving_tasks_task_id_key,
// picking_tasks_task_id_key, sales/purchase order numbers).
//
// S3.5 W4 (HR-S3-W5 C2 follow-up): articles got tenant_id in W1 but the legacy
// global UNIQUE(sku) is intentionally retained because 8+ child tables FK on
// articles.sku without tenant_id (see migration 000029). Without prefixing,
// tenant 2 signing up + auto-running SeedFarma would hit a UNIQUE violation on
// "PARA-500" and end up with an empty WMS. The prefix scopes all demo
// identifiers per tenant; user-visible NAMES are left clean ("Paracetamol
// 500mg…") so the UI still reads naturally.
//
// Format: "T<first 8 hex chars of tenantID, uppercase, no dashes>-".
// Examples:
//
//	tenantPrefix("00000000-0000-0000-0000-000000000001") -> "T00000000-"
//	tenantPrefix("a1b2c3d4-...")                          -> "TA1B2C3D4-"
//
// Empty / malformed tenantID falls back to the literal "TANON-" so the seeder
// still produces a valid (albeit collision-prone) prefix instead of panicking.
func tenantPrefix(tenantID string) string {
	clean := strings.ReplaceAll(tenantID, "-", "")
	if len(clean) < 8 {
		return "TANON-"
	}
	return "T" + strings.ToUpper(clean[:8]) + "-"
}

// prefixedSKU applies tenantPrefix to a base SKU. Centralised so the rest of
// the seeder reads naturally and any future change to the prefix scheme lives
// in one place.
func prefixedSKU(tenantID, baseSKU string) string {
	return tenantPrefix(tenantID) + baseSKU
}

// prefixedTaskID applies the tenant prefix to demo task identifiers
// (RT-DEMO-0001, PK-DEMO-0001, IN-DEMO-0001, ORD-DEMO-0001, etc.).
// receiving_tasks.task_id and picking_tasks.task_id still carry global UNIQUE
// indexes from migration 000002, so a second tenant re-running the seeder
// would otherwise collide.
func prefixedTaskID(tenantID, baseID string) string {
	return tenantPrefix(tenantID) + baseID
}

// SeedFarma seeds ~50 pharmaceutical SKUs plus tasks and movements for the given tenant.
// It is idempotent: if demo_data_seeds already has a row for (tenantID, farma-50skus), it returns nil immediately.
//
// S3.5 W4 — multi-tenant safe: every demo identifier that lives behind a still-global
// UNIQUE constraint (articles.sku, receiving_tasks.task_id, picking_tasks.task_id) is
// stamped with a per-tenant prefix derived from the first 8 hex chars of tenantID.
// Two tenants signing up via SaaS self-service therefore each get their own clean
// demo dataset instead of the second tenant hitting a duplicate-key error or
// silently inheriting tenant 1's rows. See tenantPrefix and migration 000029 for
// the full structural-debt context.
func SeedFarma(ctx context.Context, db *gorm.DB, tenantID string) error {
	// ── Idempotency check ────────────────────────────────────────────────────────
	var existing database.DemoDataSeed
	err := db.WithContext(ctx).
		Where("tenant_id = ? AND seed_name = ?", tenantID, FarmaSeedName).
		First(&existing).Error
	if err == nil {
		log.Info().Str("tenant_id", tenantID).Msg("demo seed farma-50skus already exists, skipping")
		return nil
	}

	log.Info().Str("tenant_id", tenantID).Msg("starting farma-50skus demo seed")

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Locations — returns both IDs (for FK in articles) and codes (for inventory.location)
		locationIDs, locationCodes, err := seedLocations(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("seed locations: %w", err)
		}

		// 2. Categories
		categoryIDs, err := seedCategories(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("seed categories: %w", err)
		}

		// 3. Suppliers (client type = supplier)
		supplierIDs, err := seedSuppliers(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("seed suppliers: %w", err)
		}

		// 4. Customers (client type = customer)
		customerIDs, err := seedCustomers(ctx, tx, tenantID)
		if err != nil {
			return fmt.Errorf("seed customers: %w", err)
		}

		// 5. Articles — uses locationIDs for articles.default_location_id FK
		articles, err := seedArticles(ctx, tx, tenantID, categoryIDs, locationIDs)
		if err != nil {
			return fmt.Errorf("seed articles: %w", err)
		}

		// 6. Inventory rows — uses locationCodes for inventory.location (not a FK)
		if err := seedInventory(ctx, tx, tenantID, articles, locationCodes); err != nil {
			return fmt.Errorf("seed inventory: %w", err)
		}

		// 7. Receiving tasks (10 completed + 10 partial/draft) — uses locationCodes
		if err := seedReceivingTasks(ctx, tx, tenantID, articles, locationCodes, supplierIDs); err != nil {
			return fmt.Errorf("seed receiving tasks: %w", err)
		}

		// 8. Picking tasks (8 completed + 7 partial/draft) — uses locationCodes
		if err := seedPickingTasks(ctx, tx, tenantID, articles, locationCodes, customerIDs); err != nil {
			return fmt.Errorf("seed picking tasks: %w", err)
		}

		// 9. Inventory movements (30 historical entries) — uses locationCodes
		if err := seedInventoryMovements(ctx, tx, tenantID, articles, locationCodes); err != nil {
			return fmt.Errorf("seed inventory movements: %w", err)
		}

		// 10. Mark seed as done
		meta, _ := json.Marshal(map[string]int{
			"articles":  len(articles),
			"locations": len(locationIDs),
		})
		seed := database.DemoDataSeed{
			ID:       uuid.NewString(),
			TenantID: tenantID,
			SeedName: FarmaSeedName,
			Metadata: json.RawMessage(meta),
		}
		if err := tx.Create(&seed).Error; err != nil {
			return fmt.Errorf("create demo_data_seed record: %w", err)
		}

		log.Info().Str("tenant_id", tenantID).Int("articles", len(articles)).Msg("farma-50skus seed complete")
		return nil
	})
}

// ─── locations ───────────────────────────────────────────────────────────────

// seedLocations returns (locationIDs, locationCodes, error).
// locationIDs are used for FK references in articles.default_location_id.
// locationCodes are used for inventory.location and task item location fields
// (which store codes, not IDs — known S2 debt, see feedback_estock_location_storage_inconsistency).
//
// S3.5 W4: locations now have tenant_id (migration 000032) with composite
// UNIQUE (tenant_id, location_code), so two tenants can both have "RX-A1"
// without collision. We DO NOT prefix location_code — it stays human-readable
// per tenant and the per-tenant unique index handles isolation. We DO scope
// the FirstOrCreate by tenant so a second tenant gets its own row instead of
// adopting tenant 1's location id.
func seedLocations(ctx context.Context, tx *gorm.DB, tenantID string) (ids []string, codes []string, err error) {
	locs := []struct {
		code string
		zone string
		desc string
	}{
		{"RX-A1", "RX-A", "Estante A fila 1"},
		{"RX-A2", "RX-A", "Estante A fila 2"},
		{"RX-B1", "RX-B", "Estante B fila 1"},
		{"RX-B2", "RX-B", "Estante B fila 2"},
		{"RX-C1", "RX-C", "Estante C fila 1"},
		{"RX-C2", "RX-C", "Estante C fila 2"},
		{"RX-D1", "RX-D", "Estante D fila 1"},
		{"RX-D5", "RX-D", "Estante D fila 5"},
		{"RX-E3", "RX-E", "Estante E fila 3"},
		{"RX-E10", "RX-E", "Estante E fila 10"},
	}

	for _, l := range locs {
		desc := l.desc
		loc := database.Location{
			ID:           uuid.NewString(),
			TenantID:     tenantID,
			LocationCode: l.code,
			Description:  &desc,
			Zone:         &l.zone,
			Type:         "shelf",
			IsActive:     true,
		}
		// FirstOrCreate scoped by (tenant_id, location_code) so each tenant
		// owns its own location rows. Idempotent for partial-run recovery too.
		if err := tx.WithContext(ctx).
			Where("tenant_id = ? AND location_code = ?", tenantID, l.code).
			FirstOrCreate(&loc).Error; err != nil {
			return nil, nil, fmt.Errorf("location %s: %w", l.code, err)
		}
		ids = append(ids, loc.ID)
		codes = append(codes, l.code)
	}
	return ids, codes, nil
}

// ─── categories ──────────────────────────────────────────────────────────────

func seedCategories(ctx context.Context, tx *gorm.DB, tenantID string) ([]string, error) {
	names := []string{
		"Analgésicos",
		"Antibióticos",
		"Antihistamínicos",
		"Antiinflamatorios",
		"Vitaminas",
	}
	var ids []string
	for _, name := range names {
		id := uuid.NewString()
		cat := database.Category{
			ID:       id,
			TenantID: tenantID,
			Name:     name,
			IsActive: true,
		}
		if err := tx.WithContext(ctx).
			Where("tenant_id = ? AND name = ?", tenantID, name).
			FirstOrCreate(&cat).Error; err != nil {
			return nil, fmt.Errorf("category %s: %w", name, err)
		}
		ids = append(ids, cat.ID)
	}
	return ids, nil
}

// ─── suppliers ───────────────────────────────────────────────────────────────

func seedSuppliers(ctx context.Context, tx *gorm.DB, tenantID string) ([]string, error) {
	suppliers := []struct{ code, name string }{
		{"SUP-001", "Pharma MX SA"},
		{"SUP-002", "DistriSalud CR"},
		{"SUP-003", "LabEuro SA"},
		{"SUP-004", "Genéricos Hondureños SA"},
		{"SUP-005", "MedGlobal"},
	}
	var ids []string
	for _, s := range suppliers {
		c := database.Client{
			ID:        uuid.NewString(),
			TenantID:  tenantID,
			Type:      "supplier",
			Code:      s.code,
			Name:      s.name,
			IsActive:  true,
		}
		if err := tx.WithContext(ctx).
			Where("tenant_id = ? AND code = ?", tenantID, s.code).
			FirstOrCreate(&c).Error; err != nil {
			return nil, fmt.Errorf("supplier %s: %w", s.code, err)
		}
		ids = append(ids, c.ID)
	}
	return ids, nil
}

// ─── customers ───────────────────────────────────────────────────────────────

func seedCustomers(ctx context.Context, tx *gorm.DB, tenantID string) ([]string, error) {
	customers := []struct{ code, name string }{
		{"CLI-001", "Droguería San José"},
		{"CLI-002", "Farmacia Central"},
	}
	var ids []string
	for _, cu := range customers {
		c := database.Client{
			ID:       uuid.NewString(),
			TenantID: tenantID,
			Type:     "customer",
			Code:     cu.code,
			Name:     cu.name,
			IsActive: true,
		}
		if err := tx.WithContext(ctx).
			Where("tenant_id = ? AND code = ?", tenantID, cu.code).
			FirstOrCreate(&c).Error; err != nil {
			return nil, fmt.Errorf("customer %s: %w", cu.code, err)
		}
		ids = append(ids, c.ID)
	}
	return ids, nil
}

// ─── articles ────────────────────────────────────────────────────────────────

type farmaArticle struct {
	sku        string
	name       string
	catIndex   int // index into categoryIDs
	locIndex   int // index into locationIDs (default location code)
	minQty     int
	pres       string
	unitPrice  float64
	shelfDays  int
}

var farmaArticles = []farmaArticle{
	// Analgésicos (cat 0)
	{"RX-001", "Paracetamol 500mg Tab 20s", 0, 0, 50, "Caja", 3.50, 730},
	{"RX-002", "Paracetamol 1g Tab 10s", 0, 0, 40, "Caja", 5.20, 730},
	{"RX-003", "Ibuprofeno 400mg Tab 20s", 0, 1, 30, "Caja", 4.80, 730},
	{"RX-004", "Ibuprofeno 600mg Tab 10s", 0, 1, 25, "Caja", 6.10, 730},
	{"RX-005", "Tramadol 50mg Caps 10s", 0, 2, 20, "Caja", 12.50, 730},
	{"RX-006", "Ketorolaco 30mg/1mL Inj", 0, 2, 15, "Caja", 18.00, 730},
	{"RX-007", "Metamizol 500mg Tab 20s", 0, 3, 30, "Caja", 5.50, 730},
	{"RX-008", "Codeína 30mg Tab 10s", 0, 3, 10, "Caja", 9.90, 730},
	{"RX-009", "Naproxeno 500mg Tab 20s", 0, 4, 25, "Caja", 6.50, 730},
	{"RX-010", "Aspirina 100mg Tab 30s", 0, 4, 40, "Caja", 4.20, 1095},

	// Antibióticos (cat 1)
	{"RX-011", "Amoxicilina 500mg Caps 21s", 1, 0, 30, "Caja", 8.75, 730},
	{"RX-012", "Amoxicilina 250mg/5mL Susp 100mL", 1, 0, 20, "Frasco", 6.90, 365},
	{"RX-013", "Azitromicina 500mg Tab 3s", 1, 1, 25, "Caja", 10.20, 730},
	{"RX-014", "Ciprofloxacino 500mg Tab 10s", 1, 1, 20, "Caja", 9.40, 730},
	{"RX-015", "Clindamicina 300mg Caps 16s", 1, 2, 15, "Caja", 13.80, 730},
	{"RX-016", "Metronidazol 500mg Tab 20s", 1, 2, 20, "Caja", 5.60, 730},
	{"RX-017", "Trimetoprim/Sulfametoxazol 160/800 Tab 10s", 1, 3, 15, "Caja", 4.90, 730},
	{"RX-018", "Doxiciclina 100mg Caps 10s", 1, 3, 15, "Caja", 7.30, 730},
	{"RX-019", "Eritromicina 500mg Tab 10s", 1, 4, 10, "Caja", 11.20, 730},
	{"RX-020", "Ceftriaxona 1g Inj", 1, 4, 10, "Vial", 22.50, 730},

	// Antihistamínicos (cat 2)
	{"RX-021", "Loratadina 10mg Tab 10s", 2, 5, 40, "Caja", 3.80, 730},
	{"RX-022", "Cetirizina 10mg Tab 10s", 2, 5, 35, "Caja", 4.10, 730},
	{"RX-023", "Difenhidramina 50mg Caps 10s", 2, 6, 25, "Caja", 3.60, 730},
	{"RX-024", "Fexofenadina 120mg Tab 10s", 2, 6, 20, "Caja", 7.80, 730},
	{"RX-025", "Desloratadina 5mg Tab 10s", 2, 7, 20, "Caja", 6.50, 730},
	{"RX-026", "Clorfeniramina 4mg Tab 20s", 2, 7, 30, "Caja", 2.90, 730},
	{"RX-027", "Ebastina 10mg Tab 10s", 2, 8, 15, "Caja", 8.40, 730},
	{"RX-028", "Rupatadina 10mg Tab 10s", 2, 8, 15, "Caja", 9.10, 730},
	{"RX-029", "Hidroxizina 25mg Tab 25s", 2, 9, 20, "Caja", 5.80, 730},
	{"RX-030", "Ketotifeno 1mg Tab 30s", 2, 9, 15, "Caja", 6.20, 730},

	// Antiinflamatorios (cat 3)
	{"RX-031", "Diclofenaco 50mg Tab 20s", 3, 0, 30, "Caja", 5.40, 730},
	{"RX-032", "Diclofenaco 75mg/3mL Inj", 3, 1, 20, "Caja", 14.50, 730},
	{"RX-033", "Celecoxib 200mg Caps 10s", 3, 2, 15, "Caja", 16.80, 730},
	{"RX-034", "Meloxicam 15mg Tab 10s", 3, 3, 20, "Caja", 7.90, 730},
	{"RX-035", "Piroxicam 20mg Caps 10s", 3, 4, 20, "Caja", 5.30, 730},
	{"RX-036", "Indometacina 25mg Caps 30s", 3, 5, 15, "Caja", 6.10, 730},
	{"RX-037", "Etoricoxib 90mg Tab 7s", 3, 6, 10, "Caja", 19.20, 730},
	{"RX-038", "Betametasona 0.5mg Tab 20s", 3, 7, 10, "Caja", 8.70, 730},
	{"RX-039", "Dexametasona 4mg/2mL Inj", 3, 8, 10, "Caja", 11.30, 730},
	{"RX-040", "Prednisona 20mg Tab 20s", 3, 9, 15, "Caja", 7.50, 730},

	// Vitaminas (cat 4)
	{"RX-041", "Vitamina C 500mg Tab 30s", 4, 0, 50, "Caja", 3.20, 1095},
	{"RX-042", "Vitamina D3 1000UI Tab 30s", 4, 1, 40, "Caja", 4.50, 1095},
	{"RX-043", "Complejo B Tab 30s", 4, 2, 40, "Caja", 5.10, 1095},
	{"RX-044", "Zinc 50mg Tab 30s", 4, 3, 30, "Caja", 3.90, 1095},
	{"RX-045", "Hierro 325mg Tab 30s", 4, 4, 25, "Caja", 4.80, 1095},
	{"RX-046", "Ácido Fólico 0.4mg Tab 30s", 4, 5, 30, "Caja", 2.90, 1095},
	{"RX-047", "Omega-3 1000mg Caps 60s", 4, 6, 20, "Frasco", 9.80, 1095},
	{"RX-048", "Magnesio 300mg Tab 30s", 4, 7, 20, "Caja", 5.60, 1095},
	{"RX-049", "Calcio+D3 600/400 Tab 30s", 4, 8, 20, "Caja", 6.30, 1095},
	{"RX-050", "Multivitamínico Adulto Tab 30s", 4, 9, 30, "Caja", 7.20, 1095},
}

// seedArticles inserts articles. locationIDs must be location UUIDs (FK to locations.id).
//
// S3.5 W1 (HR-S3-W5 C2 fix): articles is now tenant-scoped via composite UNIQUE
// (tenant_id, sku). FirstOrCreate now scopes by (tenant_id, sku) so each tenant
// looks up its own row instead of silently inheriting another tenant's article.
//
// S3.5 W4: SKUs are now prefixed with the tenant's short hash (see
// tenantPrefix) before insert. The legacy global UNIQUE(sku) — retained because
// 8+ child tables FK on articles.sku without tenant_id (see migration 000029) —
// would otherwise reject the second tenant's "PARA-500" / "RX-001" / etc. The
// prefix uniqueifies the SKU value across tenants while keeping the per-tenant
// composite (tenant_id, sku) intact. User-visible NAMES ("Paracetamol 500mg…")
// are kept clean so the catalog UI reads naturally.
//
// Returns the list of articles AS WRITTEN (with prefixed SKUs) so downstream
// seed steps (inventory, tasks, movements) reference the SAME prefixed SKUs.
func seedArticles(ctx context.Context, tx *gorm.DB, tenantID string, categoryIDs, locationIDs []string) ([]farmaArticle, error) {
	active := true
	written := make([]farmaArticle, 0, len(farmaArticles))
	for _, base := range farmaArticles {
		a := base                            // copy so we can mutate the SKU per-tenant
		a.sku = prefixedSKU(tenantID, base.sku)

		shelfDays := a.shelfDays
		minQty := a.minQty
		price := a.unitPrice
		catID := categoryIDs[a.catIndex%len(categoryIDs)]
		locID := locationIDs[a.locIndex%len(locationIDs)] // UUID FK — not code

		article := database.Article{
			ID:                uuid.NewString(),
			TenantID:          tenantID,
			SKU:               a.sku,
			Name:              a.name,
			Presentation:      a.pres,
			UnitPrice:         &price,
			TrackByLot:        true,
			TrackExpiration:   true,
			RotationStrategy:  "fefo",
			MinQuantity:       &minQty,
			IsActive:          &active,
			CategoryID:        &catID,
			DefaultLocationID: &locID,
			ShelfLifeInDays:   &shelfDays,
			SafetyStock:       float64(minQty) / 2,
		}
		if err := tx.WithContext(ctx).
			Where("tenant_id = ? AND sku = ?", tenantID, a.sku).
			FirstOrCreate(&article).Error; err != nil {
			return nil, fmt.Errorf("article %s: %w", a.sku, err)
		}
		written = append(written, a)
	}
	return written, nil
}

// ─── inventory ───────────────────────────────────────────────────────────────

func seedInventory(ctx context.Context, tx *gorm.DB, _ string, articles []farmaArticle, locationIDs []string) error {
	// Create ~2 inventory rows per article across different locations (100 total, capped at 100).
	count := 0
	for i, a := range articles {
		for j := 0; j < 2 && count < 100; j++ {
			locCode := locationIDs[(i+j)%len(locationIDs)]
			qty := float64(a.minQty*2 + j*10)
			price := a.unitPrice

			inv := database.Inventory{
				ID:           uuid.NewString(),
				SKU:          a.sku,
				Name:         a.name,
				Location:     locCode,
				Quantity:     qty,
				ReservedQty:  0,
				Status:       "available",
				Presentation: a.pres,
				UnitPrice:    &price,
			}
			if err := tx.WithContext(ctx).
				Where("sku = ? AND location = ?", a.sku, locCode).
				FirstOrCreate(&inv).Error; err != nil {
				return fmt.Errorf("inventory %s@%s: %w", a.sku, locCode, err)
			}
			count++
		}
	}
	return nil
}

// ─── receiving tasks ─────────────────────────────────────────────────────────

func seedReceivingTasks(ctx context.Context, tx *gorm.DB, tenantID string, articles []farmaArticle, locationIDs, supplierIDs []string) error {
	now := time.Now()

	for i := 0; i < 20; i++ {
		// First 10 are completed (past dates), last 10 are partial or draft.
		var status string
		var completedAt *time.Time
		var createdAt time.Time

		if i < 10 {
			status = "completed"
			t := now.AddDate(0, 0, -(20 - i))
			completedAt = &t
			createdAt = t.AddDate(0, 0, -1)
		} else if i < 15 {
			status = "in_progress"
			createdAt = now.AddDate(0, 0, -(i - 9))
		} else {
			status = "draft"
			createdAt = now.AddDate(0, 0, -(i - 14))
		}

		// Build 3 items per task
		var items []database.ReceivingTaskItem
		for j := 0; j < 3; j++ {
			art := articles[(i*3+j)%len(articles)]
			locCode := locationIDs[(i+j)%len(locationIDs)]
			expectedQty := float64(art.minQty)
			var acceptedQty float64
			if status == "completed" {
				acceptedQty = expectedQty
			} else if status == "in_progress" {
				acceptedQty = expectedQty / 2
			}
			items = append(items, database.ReceivingTaskItem{
				SKU:              art.sku,
				ExpectedQuantity: expectedQty,
				AcceptedQty:      acceptedQty,
				RejectedQty:      0,
				Location:         locCode,
			})
		}

		itemsJSON, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal receiving items: %w", err)
		}

		supplierID := supplierIDs[i%len(supplierIDs)]
		// S3.5 W4: receiving_tasks.task_id still has a global UNIQUE index
		// (receiving_tasks_task_id_key from migration 000002), so demo task IDs
		// must be tenant-prefixed to avoid collision when a second tenant runs
		// SeedFarma. inbound_number's UNIQUE was made per-tenant in 000019 but
		// we prefix it too for visual consistency in the UI.
		taskID := prefixedTaskID(tenantID, fmt.Sprintf("RT-DEMO-%04d", i+1))
		inboundNum := prefixedTaskID(tenantID, fmt.Sprintf("IN-DEMO-%04d", i+1))

		task := database.ReceivingTask{
			ID:          uuid.NewString(),
			TaskID:      taskID,
			InboundNumber: inboundNum,
			CreatedBy:   tenantID, // system-generated
			Status:      status,
			Priority:    priorityForIndex(i),
			Items:       json.RawMessage(itemsJSON),
			CompletedAt: completedAt,
			SupplierID:  &supplierID,
			TenantID:    tenantID,
			CreatedAt:   createdAt,
		}

		if err := tx.WithContext(ctx).
			Where("task_id = ?", taskID).
			FirstOrCreate(&task).Error; err != nil {
			return fmt.Errorf("receiving task %s: %w", taskID, err)
		}
	}
	return nil
}

// ─── picking tasks ────────────────────────────────────────────────────────────

func seedPickingTasks(ctx context.Context, tx *gorm.DB, tenantID string, articles []farmaArticle, locationIDs, customerIDs []string) error {
	now := time.Now()

	for i := 0; i < 15; i++ {
		var status string
		var completedAt *time.Time
		var createdAt time.Time

		if i < 8 {
			status = "completed"
			t := now.AddDate(0, 0, -(15 - i))
			completedAt = &t
			createdAt = t.AddDate(0, 0, -1)
		} else if i < 12 {
			status = "in_progress"
			createdAt = now.AddDate(0, 0, -(i - 7))
		} else {
			status = "pending"
			createdAt = now.AddDate(0, 0, -(i - 11))
		}

		// Build 3 items per task using PickingTaskItem shape (json stored).
		type pickingItem struct {
			SKU          string  `json:"sku"`
			Quantity     float64 `json:"quantity"`
			PickedQty    float64 `json:"picked_qty"`
			Location     string  `json:"location"`
		}
		var items []pickingItem
		for j := 0; j < 3; j++ {
			art := articles[(i*3+j+5)%len(articles)]
			locCode := locationIDs[(i+j+2)%len(locationIDs)]
			qty := float64(art.minQty)
			var picked float64
			if status == "completed" {
				picked = qty
			} else if status == "in_progress" {
				picked = qty / 2
			}
			items = append(items, pickingItem{
				SKU:       art.sku,
				Quantity:  qty,
				PickedQty: picked,
				Location:  locCode,
			})
		}

		itemsJSON, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("marshal picking items: %w", err)
		}

		customerID := customerIDs[i%len(customerIDs)]
		// S3.5 W4: picking_tasks.task_id has a global UNIQUE index
		// (picking_tasks_task_id_key from migration 000002). Tenant-prefix to
		// keep demo IDs unique across tenants.
		taskID := prefixedTaskID(tenantID, fmt.Sprintf("PK-DEMO-%04d", i+1))
		orderNum := prefixedTaskID(tenantID, fmt.Sprintf("ORD-DEMO-%04d", i+1))

		task := database.PickingTask{
			ID:          uuid.NewString(),
			TaskID:      taskID,
			OrderNumber: orderNum,
			CreatedBy:   tenantID,
			Status:      status,
			Priority:    priorityForIndex(i),
			Items:       json.RawMessage(itemsJSON),
			CompletedAt: completedAt,
			CustomerID:  &customerID,
			TenantID:    tenantID,
			CreatedAt:   createdAt,
		}

		if err := tx.WithContext(ctx).
			Where("task_id = ?", taskID).
			FirstOrCreate(&task).Error; err != nil {
			return fmt.Errorf("picking task %s: %w", taskID, err)
		}
	}
	return nil
}

// ─── inventory movements ─────────────────────────────────────────────────────

// TODO(M3 — S3.5): tenantID dropped (blank identifier _). InventoryMovement has no tenant_id
// column. Once it does (ARCH debt from C2 scope), pass and use tenantID here.
func seedInventoryMovements(ctx context.Context, tx *gorm.DB, _ string, articles []farmaArticle, locationIDs []string) error {
	now := time.Now()
	movTypes := []string{"IN", "OUT", "ADJUSTMENT", "IN", "OUT", "IN"}

	for i := 0; i < 30; i++ {
		art := articles[i%len(articles)]
		locCode := locationIDs[i%len(locationIDs)]
		movType := movTypes[i%len(movTypes)]
		qty := float64((i%10 + 1) * 5)
		remaining := float64((i%10+2) * 20)
		daysAgo := 30 - i
		createdAt := now.AddDate(0, 0, -daysAgo)
		reason := fmt.Sprintf("Demo seed movement %d", i+1)
		refType := "demo_seed"
		refID := fmt.Sprintf("DEMO-%04d", i+1)

		mov := database.InventoryMovement{
			ID:            uuid.NewString(),
			SKU:           art.sku,
			Location:      locCode,
			MovementType:  movType,
			Quantity:      qty,
			RemainingStock: remaining,
			Reason:        &reason,
			CreatedBy:     "system_seed",
			CreatedAt:     createdAt,
			ReferenceType: &refType,
			ReferenceID:   &refID,
		}

		// Use raw insert to preserve CreatedAt — GORM autoCreateTime would override it.
		if err := tx.WithContext(ctx).Exec(`
			INSERT INTO inventory_movements
				(id, sku, location, movement_type, quantity, remaining_stock, reason,
				 created_by, created_at, reference_type, reference_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT DO NOTHING`,
			mov.ID, mov.SKU, mov.Location, mov.MovementType, mov.Quantity,
			mov.RemainingStock, mov.Reason, mov.CreatedBy, mov.CreatedAt,
			mov.ReferenceType, mov.ReferenceID,
		).Error; err != nil {
			return fmt.Errorf("inventory movement %d: %w", i, err)
		}
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func priorityForIndex(i int) string {
	switch i % 3 {
	case 0:
		return "high"
	case 1:
		return "medium"
	default:
		return "low"
	}
}
