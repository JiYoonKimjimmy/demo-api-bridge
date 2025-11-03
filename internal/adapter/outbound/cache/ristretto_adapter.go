package cache

import (
	"context"
	"demo-api-bridge/internal/core/domain"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

// ristrettoAdapter는 Ristretto 기반 CacheRepository 구현체입니다.
type ristrettoAdapter struct {
	cache *ristretto.Cache
}

// RistrettoConfig는 Ristretto 캐시 설정입니다.
type RistrettoConfig struct {
	// MaxSizeMB는 캐시가 사용할 최대 메모리 크기(MB)입니다.
	MaxSizeMB int64
	// NumCounters는 Ristretto가 사용할 카운터 수입니다. (추정 항목 수의 10배 권장)
	NumCounters int64
	// BufferItems는 Get 버퍼 크기입니다.
	BufferItems int64
	// MetricsEnabled는 메트릭 수집 활성화 여부입니다.
	MetricsEnabled bool
}

// DefaultRistrettoConfig는 기본 Ristretto 설정을 반환합니다.
func DefaultRistrettoConfig() *RistrettoConfig {
	return &RistrettoConfig{
		MaxSizeMB:      1024,     // 1GB
		NumCounters:    10000000, // 10M counters (1M 항목 추정)
		BufferItems:    64,
		MetricsEnabled: true,
	}
}

// NewRistrettoAdapter는 새로운 Ristretto 어댑터를 생성합니다.
func NewRistrettoAdapter(config *RistrettoConfig) (port.CacheRepository, error) {
	if config == nil {
		config = DefaultRistrettoConfig()
	}

	// MaxCost를 바이트 단위로 변환
	maxCost := config.MaxSizeMB * 1024 * 1024

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.NumCounters,
		MaxCost:     maxCost,
		BufferItems: config.BufferItems,
		Metrics:     config.MetricsEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	return &ristrettoAdapter{
		cache: cache,
	}, nil
}

// Get은 캐시에서 값을 조회합니다.
func (r *ristrettoAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	value, found := r.cache.Get(key)
	if !found {
		return nil, domain.ErrCacheNotFound
	}

	// 타입 어설션
	bytes, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("cached value is not []byte")
	}

	return bytes, nil
}

// Set은 캐시에 값을 저장합니다.
func (r *ristrettoAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Cost는 데이터 크기로 설정 (메모리 사용량 추적)
	cost := int64(len(value))

	// Ristretto는 비동기로 Set을 처리하므로, Wait()를 호출하지 않으면 즉시 저장되지 않을 수 있음
	// 하지만 성능을 위해 Wait()는 호출하지 않음 (eventual consistency 허용)
	success := r.cache.SetWithTTL(key, value, cost, ttl)
	if !success {
		// SetWithTTL이 false를 반환하는 경우는 드물지만, 버퍼가 가득 찬 경우 발생 가능
		// 이 경우 캐시 저장 실패를 무시하고 원본 작업 계속 진행
		return fmt.Errorf("failed to set cache (buffer full or rejected)")
	}

	return nil
}

// Delete는 캐시에서 값을 삭제합니다.
func (r *ristrettoAdapter) Delete(ctx context.Context, key string) error {
	r.cache.Del(key)
	// Ristretto의 Del은 항상 성공 (에러 반환 없음)
	return nil
}

// Exists는 캐시에 키가 존재하는지 확인합니다.
func (r *ristrettoAdapter) Exists(ctx context.Context, key string) (bool, error) {
	_, found := r.cache.Get(key)
	return found, nil
}

// GetOrSet은 캐시에서 값을 조회하거나, 없으면 함수를 실행하여 저장합니다.
func (r *ristrettoAdapter) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
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
		// (로그는 상위 레이어에서 처리)
		return value, nil
	}

	return value, nil
}

// Close는 캐시를 종료하고 리소스를 정리합니다.
// Ristretto는 내부적으로 고루틴을 사용하므로 명시적으로 Close 호출 필요
func (r *ristrettoAdapter) Close() {
	if r.cache != nil {
		r.cache.Close()
	}
}

// GetMetrics는 Ristretto 캐시 메트릭을 반환합니다.
// (디버깅 및 모니터링 용도)
func (r *ristrettoAdapter) GetMetrics() *ristretto.Metrics {
	if r.cache == nil {
		return nil
	}
	return r.cache.Metrics
}
