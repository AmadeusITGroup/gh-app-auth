package cache

import (
	"testing"
	"time"
)

func TestTokenCache_GetMiss(t *testing.T) {
	cache := NewTokenCache()

	// Get non-existent key
	token, found := cache.Get("nonexistent")
	if found {
		t.Error("Expected found=false for non-existent key")
	}
	if token != "" {
		t.Errorf("Expected empty token for miss, got %q", token)
	}
}

func TestTokenCache_SetAndGetMultiple(t *testing.T) {
	cache := NewTokenCache()

	// Set multiple values
	cache.Set("key1", "token1", 1*time.Minute)
	cache.Set("key2", "token2", 1*time.Minute)
	cache.Set("key3", "token3", 1*time.Minute)

	// Get all values
	tests := []struct {
		key  string
		want string
	}{
		{"key1", "token1"},
		{"key2", "token2"},
		{"key3", "token3"},
	}

	for _, tt := range tests {
		got, found := cache.Get(tt.key)
		if !found {
			t.Errorf("Key %q not found", tt.key)
		}
		if got != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestTokenCache_DeleteMultiple(t *testing.T) {
	cache := NewTokenCache()

	// Set values
	cache.Set("key1", "token1", 1*time.Minute)
	cache.Set("key2", "token2", 1*time.Minute)

	// Delete one
	cache.Delete("key1")

	// key1 should be gone
	_, found := cache.Get("key1")
	if found {
		t.Error("key1 should be deleted")
	}

	// key2 should still exist
	_, found = cache.Get("key2")
	if !found {
		t.Error("key2 should still exist")
	}
}

func TestTokenCache_OverwriteValue(t *testing.T) {
	cache := NewTokenCache()

	// Set initial value
	cache.Set("key", "token1", 1*time.Minute)

	// Overwrite
	cache.Set("key", "token2", 1*time.Minute)

	// Should get new value
	got, found := cache.Get("key")
	if !found {
		t.Error("Key should exist")
	}
	if got != "token2" {
		t.Errorf("Expected token2, got %q", got)
	}
}

func TestTokenCache_SizeIncreases(t *testing.T) {
	cache := NewTokenCache()

	initialSize := cache.Size()

	// Add items
	cache.Set("key1", "token1", 1*time.Minute)
	size1 := cache.Size()
	if size1 <= initialSize {
		t.Error("Size should increase after adding item")
	}

	cache.Set("key2", "token2", 1*time.Minute)
	size2 := cache.Size()
	if size2 <= size1 {
		t.Error("Size should increase after adding another item")
	}
}

func TestCreateCacheKey_Consistency(t *testing.T) {
	appID := int64(123456)
	installID := int64(789012)

	// Should generate same key for same inputs
	key1 := CreateCacheKey(appID, installID)
	key2 := CreateCacheKey(appID, installID)

	if key1 != key2 {
		t.Errorf("Keys should be identical: %q vs %q", key1, key2)
	}

	// Different inputs should generate different keys
	key3 := CreateCacheKey(appID, 999999)
	if key1 == key3 {
		t.Error("Different installation IDs should generate different keys")
	}
}

func TestTokenCache_ClearResetsSize(t *testing.T) {
	cache := NewTokenCache()

	// Add items
	cache.Set("key1", "token1", 1*time.Minute)
	cache.Set("key2", "token2", 1*time.Minute)

	if cache.Size() == 0 {
		t.Error("Cache should have items")
	}

	// Clear
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Cache size after Clear() = %d, want 0", cache.Size())
	}

	// Try to get after clear
	_, found := cache.Get("key1")
	if found {
		t.Error("Should not find items after Clear()")
	}
}
