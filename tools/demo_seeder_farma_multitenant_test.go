// Package-level multi-tenant integration test for SeedFarma.
//
// S3.5 W4 (HR-S3-W5 C2 follow-up): proves that SeedFarma can be invoked for
// two distinct tenants without colliding on the still-global UNIQUE
// constraints (articles_sku_key, receiving_tasks_task_id_key,
// picking_tasks_task_id_key) and that each tenant ends up with its own
// isolated demo dataset (no cross-tenant data leak).
//
// The test piggybacks on the tenantPrefix helper rather than spinning up
// testcontainers, because:
//
//   - The full integration path is already covered by
//     repositories/signup_integration_test.go (TestSeedFarma_Idempotent +
//     TestSignup_FullFlow_InitiateAndVerify), which exercise the real DB.
//   - This package-level test belongs next to the helper it validates and
//     runs in `-short` mode (no Docker), so CI gets fast regression coverage
//     of the prefix scheme.
//
// For end-to-end multi-tenant behaviour against a real Postgres + all
// migrations, see TestSeedFarma_MultiTenantIsolation in
// repositories/signup_integration_test.go (added by the same wave).

package tools

import (
	"strings"
	"testing"
)

// TestTenantPrefix_DeterministicAndScoped checks that the prefix is stable for
// a given tenant UUID and differs across tenants — the property that prevents
// SKU collisions when two tenants run SeedFarma against the still-global
// UNIQUE(sku) index.
func TestTenantPrefix_DeterministicAndScoped(t *testing.T) {
	const (
		tenantA = "00000000-0000-0000-0000-000000000001"
		tenantB = "abcdef12-3456-7890-abcd-ef1234567890"
	)

	pA1 := tenantPrefix(tenantA)
	pA2 := tenantPrefix(tenantA)
	pB := tenantPrefix(tenantB)

	if pA1 != pA2 {
		t.Fatalf("tenantPrefix not deterministic for tenant A: %q vs %q", pA1, pA2)
	}
	if pA1 == pB {
		t.Fatalf("tenantPrefix collision between distinct tenants: A=%q B=%q", pA1, pB)
	}
	if !strings.HasPrefix(pA1, "T") || !strings.HasSuffix(pA1, "-") {
		t.Fatalf("tenantPrefix shape unexpected: %q", pA1)
	}
	if got, want := pA1, "T00000000-"; got != want {
		t.Fatalf("default tenant prefix want %q got %q", want, got)
	}
	if got, want := pB, "TABCDEF12-"; got != want {
		t.Fatalf("tenant B prefix want %q got %q", want, got)
	}
}

// TestTenantPrefix_FallbackOnMalformed makes sure malformed/empty tenant IDs
// don't panic; they get a sentinel prefix instead. Real callers should never
// hit this branch (signup creates a UUID) but the seeder is defensive.
func TestTenantPrefix_FallbackOnMalformed(t *testing.T) {
	cases := []string{"", "abc", "----", "x"}
	for _, in := range cases {
		got := tenantPrefix(in)
		if got != "TANON-" {
			t.Errorf("tenantPrefix(%q) = %q, want %q", in, got, "TANON-")
		}
	}
}

// TestPrefixedSKU_AppliesPrefix is a sanity check on the helper used in the
// articles seed loop.
func TestPrefixedSKU_AppliesPrefix(t *testing.T) {
	const tenant = "11111111-2222-3333-4444-555555555555"
	got := prefixedSKU(tenant, "RX-001")
	want := "T11111111-RX-001"
	if got != want {
		t.Fatalf("prefixedSKU = %q, want %q", got, want)
	}
}

// TestPrefixedSKU_DistinctAcrossTenants is the core regression guard for the
// W4 fix: feeding the same base SKU to two different tenants must yield two
// distinct prefixed SKUs so the global UNIQUE(sku) index is satisfied.
func TestPrefixedSKU_DistinctAcrossTenants(t *testing.T) {
	const (
		tenantA = "00000000-0000-0000-0000-000000000001"
		tenantB = "ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"
		baseSKU = "PARA-500"
	)
	skuA := prefixedSKU(tenantA, baseSKU)
	skuB := prefixedSKU(tenantB, baseSKU)
	if skuA == skuB {
		t.Fatalf("two tenants should produce distinct SKUs from same base; got both = %q", skuA)
	}
	if !strings.HasSuffix(skuA, baseSKU) || !strings.HasSuffix(skuB, baseSKU) {
		t.Fatalf("prefixed SKUs lost their base: A=%q B=%q", skuA, skuB)
	}
}

// TestPrefixedTaskID_DistinctAcrossTenants — same regression guard, but for the
// still-global UNIQUE indexes on receiving_tasks.task_id and
// picking_tasks.task_id.
func TestPrefixedTaskID_DistinctAcrossTenants(t *testing.T) {
	const (
		tenantA = "00000000-0000-0000-0000-000000000001"
		tenantB = "ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"
		baseID  = "RT-DEMO-0001"
	)
	idA := prefixedTaskID(tenantA, baseID)
	idB := prefixedTaskID(tenantB, baseID)
	if idA == idB {
		t.Fatalf("two tenants should produce distinct task IDs; got both = %q", idA)
	}
}
