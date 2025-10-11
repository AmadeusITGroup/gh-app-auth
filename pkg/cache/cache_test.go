package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTokenCache_SetAndGet(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	key := "test-key"
	token := "test-token-value"
	ttl := 5 * time.Minute

	// Test cache miss
	if _, found := cache.Get(key); found {
		t.Error("Expected cache miss, but found value")
	}

	// Set token
	cache.Set(key, token, ttl)

	// Test cache hit
	if cachedToken, found := cache.Get(key); !found {
		t.Error("Expected cache hit, but got miss")
	} else if cachedToken != token {
		t.Errorf("Expected token %s, got %s", token, cachedToken)
	}

	// Test size
	if size := cache.Size(); size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}
}

func TestTokenCache_Expiration(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	key := "expiring-key"
	token := "expiring-token"
	ttl := 100 * time.Millisecond

	// Set token with short TTL
	cache.Set(key, token, ttl)

	// Should be available immediately
	if _, found := cache.Get(key); !found {
		t.Error("Token should be available immediately after setting")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	if _, found := cache.Get(key); found {
		t.Error("Token should be expired")
	}
}

func TestTokenCache_Delete(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	key := "delete-key"
	token := "delete-token"
	ttl := 5 * time.Minute

	// Set and verify
	cache.Set(key, token, ttl)
	if _, found := cache.Get(key); !found {
		t.Error("Token should be available after setting")
	}

	// Delete and verify
	cache.Delete(key)
	if _, found := cache.Get(key); found {
		t.Error("Token should be deleted")
	}

	// Size should be 0
	if size := cache.Size(); size != 0 {
		t.Errorf("Expected cache size 0 after delete, got %d", size)
	}
}

func TestTokenCache_Clear(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	// Add multiple tokens
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		token := fmt.Sprintf("token-%d", i)
		cache.Set(key, token, 5*time.Minute)
	}

	// Verify size
	if size := cache.Size(); size != 5 {
		t.Errorf("Expected cache size 5, got %d", size)
	}

	// Clear cache
	cache.Clear()

	// Verify empty
	if size := cache.Size(); size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", size)
	}

	// Verify no tokens are retrievable
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		if _, found := cache.Get(key); found {
			t.Errorf("Token %s should not be found after clear", key)
		}
	}
}

func TestTokenCache_ConcurrentAccess(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines doing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				token := fmt.Sprintf("concurrent-token-%d-%d", id, j)

				// Set token
				cache.Set(key, token, 1*time.Minute)

				// Get token
				if cachedToken, found := cache.Get(key); found {
					if cachedToken != token {
						t.Errorf("Expected token %s, got %s", token, cachedToken)
					}
				}

				// Delete token (every other operation)
				if j%2 == 0 {
					cache.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()

	// Cache should have approximately half the tokens (those not deleted)
	size := cache.Size()
	expectedMin := numGoroutines * numOperations / 4 // At least 25% should remain
	expectedMax := numGoroutines * numOperations     // At most all could remain

	if size < expectedMin || size > expectedMax {
		t.Errorf("Expected cache size between %d and %d, got %d", expectedMin, expectedMax, size)
	}
}

func TestTokenCache_GetStats(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	// Add some valid tokens
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("valid-key-%d", i)
		token := fmt.Sprintf("valid-token-%d", i)
		cache.Set(key, token, 5*time.Minute)
	}

	// Add some expired tokens
	for i := 0; i < 2; i++ {
		key := fmt.Sprintf("expired-key-%d", i)
		token := fmt.Sprintf("expired-token-%d", i)
		cache.Set(key, token, 1*time.Millisecond)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	stats := cache.GetStats()

	if stats.TotalTokens != 5 {
		t.Errorf("Expected total tokens 5, got %d", stats.TotalTokens)
	}

	if stats.ValidTokens != 3 {
		t.Errorf("Expected valid tokens 3, got %d", stats.ValidTokens)
	}

	if stats.ExpiredTokens != 2 {
		t.Errorf("Expected expired tokens 2, got %d", stats.ExpiredTokens)
	}
}

func TestTokenCache_Cleanup(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	// Add tokens with different expiration times
	cache.Set("short-lived", "token1", 50*time.Millisecond)
	cache.Set("long-lived", "token2", 5*time.Minute)

	// Verify both are initially present
	if size := cache.Size(); size != 2 {
		t.Errorf("Expected initial size 2, got %d", size)
	}

	// Wait for short-lived token to expire
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	cache.cleanup()

	// Only long-lived token should remain
	if size := cache.Size(); size != 1 {
		t.Errorf("Expected size 1 after cleanup, got %d", size)
	}

	// Verify the correct token remains
	if _, found := cache.Get("long-lived"); !found {
		t.Error("Long-lived token should still be available")
	}

	if _, found := cache.Get("short-lived"); found {
		t.Error("Short-lived token should be cleaned up")
	}
}

func TestCreateCacheKey(t *testing.T) {
	tests := []struct {
		name           string
		appID          int64
		installationID int64
		expected       string
	}{
		{
			name:           "positive IDs",
			appID:          12345,
			installationID: 67890,
			expected:       "app_12345_inst_67890",
		},
		{
			name:           "zero IDs",
			appID:          0,
			installationID: 0,
			expected:       "app_0_inst_0",
		},
		{
			name:           "large IDs",
			appID:          999999999999,
			installationID: 888888888888,
			expected:       "app_999999999999_inst_888888888888",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateCacheKey(tt.appID, tt.installationID)
			if result != tt.expected {
				t.Errorf("CreateCacheKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTokenCache_OverwriteExisting(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	key := "overwrite-key"
	token1 := "first-token"
	token2 := "second-token"
	ttl := 5 * time.Minute

	// Set first token
	cache.Set(key, token1, ttl)
	if cachedToken, found := cache.Get(key); !found || cachedToken != token1 {
		t.Errorf("Expected first token %s, got %s (found: %v)", token1, cachedToken, found)
	}

	// Overwrite with second token
	cache.Set(key, token2, ttl)
	if cachedToken, found := cache.Get(key); !found || cachedToken != token2 {
		t.Errorf("Expected second token %s, got %s (found: %v)", token2, cachedToken, found)
	}

	// Size should still be 1
	if size := cache.Size(); size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}
}

func TestTokenCache_DeleteNonExistent(t *testing.T) {
	cache := NewTokenCache()
	defer cache.Clear()

	// Deleting non-existent key should not panic
	cache.Delete("non-existent-key")

	// Size should be 0
	if size := cache.Size(); size != 0 {
		t.Errorf("Expected cache size 0, got %d", size)
	}
}
