package cache

import (
	"context"
	"time"
)

// NoopCache is a Cache that does nothing.
// Used as a startup fallback when Redis is unavailable, and in unit tests.
type NoopCache struct{}

func (NoopCache) Get(_ context.Context, _ string) (string, bool, error) { return "", false, nil }
func (NoopCache) Set(_ context.Context, _, _ string, _ time.Duration) error { return nil }
func (NoopCache) Delete(_ context.Context, _ ...string) error              { return nil }
func (NoopCache) Ping(_ context.Context) error                             { return nil }
