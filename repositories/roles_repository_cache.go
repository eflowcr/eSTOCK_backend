package repositories

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/eflowcr/eSTOCK_backend/ports"
)

type cachedPerms struct {
	perms []byte
	until time.Time
}

// RolesRepositoryCache wraps a RolesRepository and caches GetRolePermissions by role ID
// with a TTL. UpdatePermissions invalidates the cache for that role so changes apply immediately.
type RolesRepositoryCache struct {
	repo ports.RolesRepository
	ttl  time.Duration
	mu   sync.RWMutex
	m    map[string]cachedPerms
}

// NewRolesRepositoryCache returns a caching wrapper around the given repository.
func NewRolesRepositoryCache(repo ports.RolesRepository, ttl time.Duration) *RolesRepositoryCache {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &RolesRepositoryCache{repo: repo, ttl: ttl, m: make(map[string]cachedPerms)}
}

var _ ports.RolesRepository = (*RolesRepositoryCache)(nil)

func (c *RolesRepositoryCache) GetRolePermissions(ctx context.Context, roleID string) ([]byte, error) {
	c.mu.RLock()
	if p, ok := c.m[roleID]; ok && time.Now().Before(p.until) {
		c.mu.RUnlock()
		return p.perms, nil
	}
	c.mu.RUnlock()

	perms, err := c.repo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.m[roleID] = cachedPerms{perms: perms, until: time.Now().Add(c.ttl)}
	c.mu.Unlock()
	return perms, nil
}

func (c *RolesRepositoryCache) List(ctx context.Context) ([]ports.RoleEntry, error) {
	return c.repo.List(ctx)
}

func (c *RolesRepositoryCache) GetByID(ctx context.Context, roleID string) (*ports.RoleEntry, error) {
	return c.repo.GetByID(ctx, roleID)
}

func (c *RolesRepositoryCache) UpdatePermissions(ctx context.Context, roleID string, permissions json.RawMessage) error {
	err := c.repo.UpdatePermissions(ctx, roleID, permissions)
	if err != nil {
		return err
	}
	c.mu.Lock()
	delete(c.m, roleID)
	c.mu.Unlock()
	return nil
}
