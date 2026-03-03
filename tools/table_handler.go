package tools

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TableConfig defines how a domain entity is exposed via the generic table handler.
type TableConfig struct {
	EntityName   string            // for logging/messages, e.g. "articles"
	FromClause   string            // e.g. "articles a"
	AllowedCols  map[string]string // frontend column id -> SQL column, e.g. "name" -> "a.name"
	SearchFields []string          // columns for ILIKE search

	DefaultWhere string   // optional static WHERE fragment (without "WHERE")
	SelectFields string   // SELECT clause for list/export, e.g. "a.id, a.sku, a.name"
	CSVFields    []string // columns in SELECT order for CSV (keys from queryRows map)
	CSVHeaders   []string // header row for CSV

	DefaultSortBy  string // column id (key in AllowedCols)
	DefaultSortDir string // "asc" or "desc"

	NoPagination bool // if true, ignore page/per_page and return all rows
}

// tableFilter is one filter in ?filters=[{"id":"is_active","op":"eq","value":true}].
type tableFilter struct {
	ID    string      `json:"id"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

// GenericListHandler returns a Gin handler that lists rows for the given config using the pgx pool.
// Response:
//   { data: [...], pagination: { page, perPage, total, filteredTotal } }
func GenericListHandler(pool *pgxpool.Pool, cfg TableConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			ResponseInternal(c, "GenericListHandler", "Base de datos no disponible", "generic_list_db_unavailable")
			return
		}

		page, perPage, err := parsePageParams(c)
		if err != nil {
			ResponseBadRequest(c, "GenericListHandler", err.Error(), "generic_list_invalid_pagination")
			return
		}

		sortBy, sortDir := parseSortParams(c, cfg)
		search := strings.TrimSpace(c.Query("q"))
		filters, err := parseFilters(c.Query("filters"))
		if err != nil {
			ResponseBadRequest(c, "GenericListHandler", "Parámetro 'filters' inválido", "generic_list_invalid_filters")
			return
		}

		limit := perPage
		offset := (page - 1) * perPage
		if cfg.NoPagination {
			limit = 0
			offset = 0
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		whereSQL, args := buildWhereClause(cfg, search, filters)
		orderSQL := buildOrderBy(cfg, sortBy, sortDir)

		total, err := countTotal(ctx, pool, cfg, "")
		if err != nil {
			ResponseInternal(c, "GenericListHandler", "Error al obtener el total de registros", "generic_list_total_error")
			return
		}

		filteredTotal, err := countTotal(ctx, pool, cfg, whereSQL, args...)
		if err != nil {
			ResponseInternal(c, "GenericListHandler", "Error al obtener el total filtrado", "generic_list_filtered_total_error")
			return
		}

		rows, err := queryRows(ctx, pool, cfg, whereSQL, orderSQL, limit, offset, args...)
		if err != nil {
			ResponseInternal(c, "GenericListHandler", "Error al obtener los registros", "generic_list_query_error")
			return
		}

		payload := gin.H{
			"data": rows,
			"pagination": gin.H{
				"page":          page,
				"perPage":       perPage,
				"total":         total,
				"filteredTotal": filteredTotal,
			},
		}
		ResponseOK(c, "GenericListHandler", fmt.Sprintf("%s listados con éxito", strings.Title(cfg.EntityName)), "generic_list_success", payload, false, "")
	}
}

// GenericExportHandler returns a Gin handler that exports CSV with the same filters/search as the list handler.
func GenericExportHandler(pool *pgxpool.Pool, cfg TableConfig, filename string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			ResponseInternal(c, "GenericExportHandler", "Base de datos no disponible", "generic_export_db_unavailable")
			return
		}

		search := strings.TrimSpace(c.Query("q"))
		filters, err := parseFilters(c.Query("filters"))
		if err != nil {
			ResponseBadRequest(c, "GenericExportHandler", "Parámetro 'filters' inválido", "generic_export_invalid_filters")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		whereSQL, args := buildWhereClause(cfg, search, filters)
		orderSQL := buildOrderBy(cfg, cfg.DefaultSortBy, cfg.DefaultSortDir)

		rows, err := queryRows(ctx, pool, cfg, whereSQL, orderSQL, 0, 0, args...)
		if err != nil {
			ResponseInternal(c, "GenericExportHandler", "Error al exportar registros", "generic_export_query_error")
			return
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		w := csv.NewWriter(c.Writer)
		if len(cfg.CSVHeaders) > 0 {
			_ = w.Write(cfg.CSVHeaders)
		}

		for _, row := range rows {
			record := make([]string, len(cfg.CSVFields))
			for i, field := range cfg.CSVFields {
				val := row[field]
				if val == nil {
					record[i] = ""
					continue
				}
				record[i] = fmt.Sprintf("%v", val)
			}
			_ = w.Write(record)
		}
		w.Flush()
	}
}

func parsePageParams(c *gin.Context) (int, int, error) {
	pageStr := c.DefaultQuery("page", "1")
	perPageStr := c.DefaultQuery("per_page", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, 0, fmt.Errorf("page inválido")
	}
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 {
		return 0, 0, fmt.Errorf("per_page inválido")
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage, nil
}

func parseSortParams(c *gin.Context, cfg TableConfig) (string, string) {
	sortBy := c.DefaultQuery("sort_by", cfg.DefaultSortBy)
	sortDir := strings.ToLower(c.DefaultQuery("sort_dir", cfg.DefaultSortDir))
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = cfg.DefaultSortDir
	}
	if _, ok := cfg.AllowedCols[sortBy]; !ok {
		sortBy = cfg.DefaultSortBy
	}
	return sortBy, sortDir
}

func parseFilters(raw string) ([]tableFilter, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var filters []tableFilter
	if err := json.Unmarshal([]byte(raw), &filters); err != nil {
		return nil, err
	}
	return filters, nil
}

func buildWhereClause(cfg TableConfig, search string, filters []tableFilter) (string, []interface{}) {
	clauses := []string{}
	args := []interface{}{}

	if cfg.DefaultWhere != "" {
		clauses = append(clauses, cfg.DefaultWhere)
	}

	// Search across SearchFields with ILIKE
	if search != "" && len(cfg.SearchFields) > 0 {
		orParts := make([]string, 0, len(cfg.SearchFields))
		for range cfg.SearchFields {
			orParts = append(orParts, fmt.Sprintf("%s ILIKE $%d", "?", len(args)+1))
			args = append(args, "%"+search+"%")
		}
		for i, col := range cfg.SearchFields {
			orParts[i] = strings.Replace(orParts[i], "?", col, 1)
		}
		clauses = append(clauses, "("+strings.Join(orParts, " OR ")+")")
	}

	for _, f := range filters {
		col, ok := cfg.AllowedCols[f.ID]
		if !ok {
			continue
		}
		op := strings.ToLower(f.Op)
		switch op {
		case "eq", "neq", "contains", "startswith":
			args = append(args, f.Value)
			placeholder := fmt.Sprintf("$%d", len(args))
			switch op {
			case "eq":
				clauses = append(clauses, fmt.Sprintf("%s = %s", col, placeholder))
			case "neq":
				clauses = append(clauses, fmt.Sprintf("%s <> %s", col, placeholder))
			case "contains":
				clauses = append(clauses, fmt.Sprintf("%s ILIKE %s", col, placeholder))
				args[len(args)-1] = "%"+fmt.Sprint(f.Value)+"%"
			case "startswith":
				clauses = append(clauses, fmt.Sprintf("%s ILIKE %s", col, placeholder))
				args[len(args)-1] = fmt.Sprint(f.Value) + "%"
			}
		case "istrue":
			clauses = append(clauses, fmt.Sprintf("%s = TRUE", col))
		case "isfalse":
			clauses = append(clauses, fmt.Sprintf("%s = FALSE", col))
		default:
			continue
		}
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func buildOrderBy(cfg TableConfig, sortBy, sortDir string) string {
	col, ok := cfg.AllowedCols[sortBy]
	if !ok {
		col = cfg.AllowedCols[cfg.DefaultSortBy]
	}
	return fmt.Sprintf("ORDER BY %s %s", col, strings.ToUpper(sortDir))
}

func countTotal(ctx context.Context, pool *pgxpool.Pool, cfg TableConfig, whereSQL string, args ...interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", cfg.FromClause)
	if whereSQL != "" {
		query += " " + whereSQL
	}
	row := pool.QueryRow(ctx, query, args...)
	var total int64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func queryRows(ctx context.Context, pool *pgxpool.Pool, cfg TableConfig, whereSQL, orderSQL string, limit, offset int, args ...interface{}) ([]map[string]interface{}, error) {
	selectClause := cfg.SelectFields
	if strings.TrimSpace(selectClause) == "" {
		selectClause = "*"
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectClause, cfg.FromClause)
	if whereSQL != "" {
		query += " " + whereSQL
	}
	if orderSQL != "" {
		query += " " + orderSQL
	}
	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	result := []map[string]interface{}{}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		rowMap := make(map[string]interface{}, len(values))
		for i, fd := range fieldDescriptions {
			rowMap[string(fd.Name)] = values[i]
		}
		result = append(result, rowMap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

