package cache

import (
	"context"
	"testing"
	"time"
)

func TestNoopCache(t *testing.T) {
	c := NoopCache{}
	ctx := context.Background()

	if _, ok, err := c.Get(ctx, "key"); ok || err != nil {
		t.Errorf("NoopCache.Get: expected (false, nil), got (%v, %v)", ok, err)
	}
	if err := c.Set(ctx, "key", "val", time.Minute); err != nil {
		t.Errorf("NoopCache.Set: unexpected error: %v", err)
	}
	if err := c.Delete(ctx, "key"); err != nil {
		t.Errorf("NoopCache.Delete: unexpected error: %v", err)
	}
	if err := c.Ping(ctx); err != nil {
		t.Errorf("NoopCache.Ping: unexpected error: %v", err)
	}
}

func TestCacheKeys(t *testing.T) {
	if got := SKUKey("MED-001"); got != "inv:sku:MED-001" {
		t.Errorf("SKUKey: expected inv:sku:MED-001, got %q", got)
	}
	if got := LowStockKey(); got != "inv:low-stock" {
		t.Errorf("LowStockKey: expected inv:low-stock, got %q", got)
	}
}
