package cache

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRistrettoAdapter(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		adapter, err := NewRistrettoAdapter(nil)
		require.NoError(t, err)
		require.NotNil(t, adapter)

		// Close 테스트
		ristrettoAdapter := adapter.(*ristrettoAdapter)
		ristrettoAdapter.Close()
	})

	t.Run("custom config", func(t *testing.T) {
		config := &RistrettoConfig{
			MaxSizeMB:      100,
			NumCounters:    1000000,
			BufferItems:    32,
			MetricsEnabled: true,
		}

		adapter, err := NewRistrettoAdapter(config)
		require.NoError(t, err)
		require.NotNil(t, adapter)

		ristrettoAdapter := adapter.(*ristrettoAdapter)
		defer ristrettoAdapter.Close()
	})
}

func TestRistrettoAdapter_SetAndGet(t *testing.T) {
	adapter, err := NewRistrettoAdapter(nil)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()

	t.Run("set and get value", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := 10 * time.Second

		// Set
		err := adapter.Set(ctx, key, value, ttl)
		assert.NoError(t, err)

		// Ristretto는 비동기로 Set을 처리하므로 약간의 대기 필요
		time.Sleep(10 * time.Millisecond)

		// Get
		retrieved, err := adapter.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		key := "non-existent-key"

		retrieved, err := adapter.Get(ctx, key)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrCacheNotFound, err)
		assert.Nil(t, retrieved)
	})
}

func TestRistrettoAdapter_Delete(t *testing.T) {
	adapter, err := NewRistrettoAdapter(nil)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")
	ttl := 10 * time.Second

	// Set
	err = adapter.Set(ctx, key, value, ttl)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// Verify exists
	retrieved, err := adapter.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Delete
	err = adapter.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify deleted
	retrieved, err = adapter.Get(ctx, key)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrCacheNotFound, err)
}

func TestRistrettoAdapter_Exists(t *testing.T) {
	adapter, err := NewRistrettoAdapter(nil)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")
	ttl := 10 * time.Second

	// Non-existent key
	exists, err := adapter.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Set
	err = adapter.Set(ctx, key, value, ttl)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// Existent key
	exists, err = adapter.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRistrettoAdapter_GetOrSet(t *testing.T) {
	adapter, err := NewRistrettoAdapter(nil)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()

	t.Run("cache miss - execute function", func(t *testing.T) {
		key := "test-key-getorset"
		expectedValue := []byte("function-result")
		ttl := 10 * time.Second

		fnCalled := false
		fn := func() ([]byte, error) {
			fnCalled = true
			return expectedValue, nil
		}

		// First call - cache miss
		result, err := adapter.GetOrSet(ctx, key, ttl, fn)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
		assert.True(t, fnCalled, "function should be called on cache miss")

		// Wait for async set to complete
		time.Sleep(10 * time.Millisecond)

		// Second call - cache hit
		fnCalled = false
		result, err = adapter.GetOrSet(ctx, key, ttl, fn)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, result)
		assert.False(t, fnCalled, "function should not be called on cache hit")
	})

	t.Run("function returns error", func(t *testing.T) {
		key := "test-key-error"
		ttl := 10 * time.Second

		fn := func() ([]byte, error) {
			return nil, assert.AnError
		}

		result, err := adapter.GetOrSet(ctx, key, ttl, fn)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRistrettoAdapter_TTL(t *testing.T) {
	adapter, err := NewRistrettoAdapter(nil)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()

	key := "test-key-ttl"
	value := []byte("test-value")
	ttl := 100 * time.Millisecond

	// Set with short TTL
	err = adapter.Set(ctx, key, value, ttl)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// Verify exists
	retrieved, err := adapter.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Verify expired
	retrieved, err = adapter.Get(ctx, key)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrCacheNotFound, err)
}

func TestRistrettoAdapter_GetMetrics(t *testing.T) {
	config := &RistrettoConfig{
		MaxSizeMB:      10,
		NumCounters:    100000,
		BufferItems:    64,
		MetricsEnabled: true,
	}

	adapter, err := NewRistrettoAdapter(config)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()

	// Perform some operations
	key := "test-key-metrics"
	value := []byte("test-value")
	ttl := 10 * time.Second

	err = adapter.Set(ctx, key, value, ttl)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	_, _ = adapter.Get(ctx, key)           // Hit
	_, _ = adapter.Get(ctx, "missing-key") // Miss

	// Get metrics
	ristrettoAdapter := adapter.(*ristrettoAdapter)
	metrics := ristrettoAdapter.GetMetrics()
	require.NotNil(t, metrics)

	// Verify metrics are collected
	assert.Greater(t, metrics.Hits(), uint64(0), "should have cache hits")
	assert.Greater(t, metrics.Misses(), uint64(0), "should have cache misses")
}

func TestRistrettoAdapter_LargeBatch(t *testing.T) {
	adapter, err := NewRistrettoAdapter(nil)
	require.NoError(t, err)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()
	ttl := 10 * time.Second

	// Set multiple keys
	numKeys := 100
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		err := adapter.Set(ctx, key, value, ttl)
		assert.NoError(t, err)
	}

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify all keys exist
	foundCount := 0
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		if _, err := adapter.Get(ctx, key); err == nil {
			foundCount++
		}
	}

	// Ristretto는 비동기 처리로 인해 일부 키가 누락될 수 있음
	// 최소 90% 이상의 키가 존재하면 성공으로 간주
	assert.GreaterOrEqual(t, foundCount, numKeys*9/10,
		"at least 90%% of keys should be found")
}

// Benchmark: Ristretto vs Redis 성능 비교를 위한 벤치마크
func BenchmarkRistrettoAdapter_Set(b *testing.B) {
	adapter, _ := NewRistrettoAdapter(nil)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()
	ttl := 10 * time.Second
	value := []byte("benchmark-value-1234567890")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		_ = adapter.Set(ctx, key, value, ttl)
	}
}

func BenchmarkRistrettoAdapter_Get(b *testing.B) {
	adapter, _ := NewRistrettoAdapter(nil)
	defer adapter.(*ristrettoAdapter).Close()

	ctx := context.Background()
	ttl := 10 * time.Second
	key := "bench-key"
	value := []byte("benchmark-value-1234567890")

	_ = adapter.Set(ctx, key, value, ttl)
	time.Sleep(10 * time.Millisecond) // Wait for async set

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = adapter.Get(ctx, key)
	}
}
