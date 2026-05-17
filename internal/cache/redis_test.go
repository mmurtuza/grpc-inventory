package cache

import (
	"testing"
)

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("invalid-url")
	if err == nil {
		t.Error("expected error for invalid redis url, got nil")
	}
}
