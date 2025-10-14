package cache

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisAdapter는 Redis 기반 CacheRepository 구현체입니다.
type redisAdapter struct {
	client *redis.Client
}

// NewRedisAdapter는 새로운 Redis 어댑터를 생성합니다.
func NewRedisAdapter(addr, password string, db int) (port.CacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisAdapter{
		client: client,
	}, nil
}

// NewRedisAdapterWithClient는 기존 Redis 클라이언트로 어댑터를 생성합니다.
func NewRedisAdapterWithClient(client *redis.Client) port.CacheRepository {
	return &redisAdapter{
		client: client,
	}
}

// Get은 캐시에서 값을 조회합니다.
func (r *redisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrCacheNotFound
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	return []byte(val), nil
}

// Set은 캐시에 값을 저장합니다.
func (r *redisAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete는 캐시에서 값을 삭제합니다.
func (r *redisAdapter) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// Exists는 캐시에 키가 존재하는지 확인합니다.
func (r *redisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cache existence: %w", err)
	}

	return count > 0, nil
}

// GetOrSet은 캐시에서 값을 조회하거나, 없으면 함수를 실행하여 저장합니다.
func (r *redisAdapter) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	// 먼저 캐시에서 조회
	if value, err := r.Get(ctx, key); err == nil {
		return value, nil
	}

	// 캐시에 없으면 함수 실행
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// 결과를 캐시에 저장
	if err := r.Set(ctx, key, value, ttl); err != nil {
		// 캐시 저장 실패는 무시하고 원본 값을 반환
		return value, nil
	}

	return value, nil
}
