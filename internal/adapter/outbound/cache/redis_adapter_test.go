package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisAdapter_GetSet(t *testing.T) {
	// 실제 Redis가 없으므로 Mock 테스트는 생략
	// 실제 환경에서는 Redis 인스턴스가 필요
	t.Skip("Redis instance required")
}

func TestRedisAdapter_Delete(t *testing.T) {
	t.Skip("Redis instance required")
}

func TestRedisAdapter_Exists(t *testing.T) {
	t.Skip("Redis instance required")
}

func TestRedisAdapter_GetOrSet(t *testing.T) {
	t.Skip("Redis instance required")
}

func TestRedisAdapter_Ping(t *testing.T) {
	t.Skip("Redis instance required")
}

// Mock Redis 클라이언트를 사용한 테스트 (실제 구현은 복잡하므로 생략)
func TestRedisAdapter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// 실제 Redis 연결 테스트
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // 테스트용 DB
	})

	// Redis 연결 확인
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis instance not available: %v", err)
	}

	adapter := NewRedisAdapterWithClient(client)

	// 기본 CRUD 테스트
	key := "test_key"
	value := []byte("test_value")

	// Set
	if err := adapter.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	retrieved, err := adapter.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	// Exists
	exists, err := adapter.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("Key should exist")
	}

	// Delete
	if err := adapter.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get after delete
	_, err = adapter.Get(ctx, key)
	if err == nil {
		t.Error("Expected error after delete")
	}

	// Cleanup
	client.Close()
}
