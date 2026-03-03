// Integration tests for ArticlesRepositorySQLC. Requires Docker (testcontainers).
// Run from backend dir: go test -v ./repositories/... -run TestArticlesRepositorySQLC

package repositories

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/db/sqlc"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDB(t *testing.T) (connStr string, cleanup func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	cleanup = func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	connStr, err = postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	return connStr, cleanup
}

func runMigrations(t *testing.T, connStr string) {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	migrationPath := filepath.Join(dir, "..", "db", "migrations")
	migrationURL := "file://" + filepath.ToSlash(migrationPath)

	err := tools.RunMigrations(migrationURL, connStr)
	require.NoError(t, err, "migrations failed")
}

func newTestRepo(t *testing.T, connStr string) *ArticlesRepositorySQLC {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "failed to create pool")
	t.Cleanup(func() { pool.Close() })

	queries := sqlc.New(pool)
	return NewArticlesRepositorySQLC(queries)
}

func TestArticlesRepositorySQLC_ListAndCreate(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	// List empty
	list, resp := repo.GetAllArticles()
	require.Nil(t, resp)
	assert.Empty(t, list)

	// Create
	data := &requests.Article{
		SKU:          "TEST-SKU-001",
		Name:         "Test Article",
		Presentation: "unit",
	}
	resp = repo.CreateArticle(data)
	require.Nil(t, resp)

	// List has one
	list, resp = repo.GetAllArticles()
	require.Nil(t, resp)
	require.Len(t, list, 1)
	assert.Equal(t, "TEST-SKU-001", list[0].SKU)
	assert.Equal(t, "Test Article", list[0].Name)
}

func TestArticlesRepositorySQLC_GetByIDAndBySku(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	// Create
	data := &requests.Article{
		SKU:          "TEST-SKU-002",
		Name:         "Get Test",
		Presentation: "unit",
	}
	resp := repo.CreateArticle(data)
	require.Nil(t, resp)

	list, _ := repo.GetAllArticles()
	require.Len(t, list, 1)
	id := list[0].ID

	// Get by ID
	art, resp := repo.GetArticleByID(id)
	require.Nil(t, resp)
	require.NotNil(t, art)
	assert.Equal(t, id, art.ID)
	assert.Equal(t, "TEST-SKU-002", art.SKU)

	// Get by SKU
	art, resp = repo.GetBySku("TEST-SKU-002")
	require.Nil(t, resp)
	require.NotNil(t, art)
	assert.Equal(t, "TEST-SKU-002", art.SKU)
}

func TestArticlesRepositorySQLC_GetByID_NotFound(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	art, resp := repo.GetArticleByID(99999)
	require.Nil(t, art)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
	assert.Equal(t, "Artículo no encontrado", resp.Message)
}

func TestArticlesRepositorySQLC_GetBySku_NotFound(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	art, resp := repo.GetBySku("NONEXISTENT-SKU")
	require.Nil(t, art)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}

func TestArticlesRepositorySQLC_CreateDuplicate_Conflict(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	data := &requests.Article{
		SKU:          "DUP-SKU",
		Name:         "First",
		Presentation: "unit",
	}
	resp := repo.CreateArticle(data)
	require.Nil(t, resp)

	// Duplicate SKU
	data2 := &requests.Article{
		SKU:          "DUP-SKU",
		Name:         "Second",
		Presentation: "unit",
	}
	resp = repo.CreateArticle(data2)
	require.NotNil(t, resp)
	assert.True(t, resp.Handled)
	assert.Equal(t, responses.StatusConflict, resp.StatusCode)
	assert.Contains(t, resp.Message, "SKU")
}

func TestArticlesRepositorySQLC_UpdateAndDelete(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	// Create
	data := &requests.Article{
		SKU:          "UPDATE-SKU",
		Name:         "Original",
		Presentation: "unit",
	}
	resp := repo.CreateArticle(data)
	require.Nil(t, resp)

	list, _ := repo.GetAllArticles()
	require.Len(t, list, 1)
	id := list[0].ID

	// Update
	updated := &requests.Article{
		SKU:          "UPDATE-SKU",
		Name:         "Updated Name",
		Presentation: "unit",
	}
	art, resp := repo.UpdateArticle(id, updated)
	require.Nil(t, resp)
	require.NotNil(t, art)
	assert.Equal(t, "Updated Name", art.Name)

	// Delete
	resp = repo.DeleteArticle(id)
	require.Nil(t, resp)

	// Get returns 404
	art, resp = repo.GetArticleByID(id)
	require.Nil(t, art)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}

func TestArticlesRepositorySQLC_Update_NotFound(t *testing.T) {
	connStr, cleanup := setupTestDB(t)
	defer cleanup()
	runMigrations(t, connStr)
	repo := newTestRepo(t, connStr)

	data := &requests.Article{
		SKU:          "X",
		Name:         "X",
		Presentation: "unit",
	}
	art, resp := repo.UpdateArticle(99999, data)
	require.Nil(t, art)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}
