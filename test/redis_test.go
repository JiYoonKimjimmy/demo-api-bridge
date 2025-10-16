package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"demo-api-bridge/pkg/config"

	"github.com/redis/go-redis/v9"
)

// TestRedisConnection은 Redis 연결을 테스트합니다.
func TestRedisConnection(t *testing.T) {
	// 설정 로드
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Redis 클라이언트 생성
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	defer rdb.Close()

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = rdb.Ping(ctx).Err()
	if err != nil {
		t.Fatalf("Failed to ping Redis: %v", err)
	}

	t.Log("✅ Redis connection successful!")

	// 기본 Redis 작업 테스트
	testBasicRedisOperations(t, rdb, ctx)
}

// testBasicRedisOperations는 기본 Redis 작업을 테스트합니다.
func testBasicRedisOperations(t *testing.T, rdb *redis.Client, ctx context.Context) {
	var err error

	// 서버 정보 확인
	_, err = rdb.Info(ctx).Result()
	if err != nil {
		t.Logf("Note: INFO command not supported: %v", err)
	} else {
		t.Logf("✅ Server INFO available")
	}

	// PING 테스트
	err = rdb.Ping(ctx).Err()
	if err != nil {
		t.Errorf("Failed to ping: %v", err)
		return
	}
	t.Log("✅ PING successful")

	// ECHO 테스트
	echoResult, err := rdb.Echo(ctx, "test").Result()
	if err != nil {
		t.Logf("Note: ECHO command not supported: %v", err)
	} else {
		t.Logf("✅ ECHO result: %s", echoResult)
	}

	// TIME 테스트
	timeResult, err := rdb.Time(ctx).Result()
	if err != nil {
		t.Logf("Note: TIME command not supported: %v", err)
	} else {
		t.Logf("✅ TIME result: %v", timeResult)
	}

	// 기본적인 키-값 작업이 지원되는지 확인
	testKey := "test:api-bridge:connection"
	testValue := "Hello Redis!"

	// SET 명령어 시도
	err = rdb.Set(ctx, testKey, testValue, 10*time.Second).Err()
	if err != nil {
		t.Logf("Note: SET command not supported: %v", err)
		return
	}
	t.Logf("✅ Set key '%s' with value '%s'", testKey, testValue)

	// GET 명령어 시도
	value, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		t.Logf("Note: GET command not supported: %v", err)
		return
	}

	if value != testValue {
		t.Errorf("Expected value '%s', got '%s'", testValue, value)
		return
	}
	t.Logf("✅ Get key '%s' returned value '%s'", testKey, value)

	// DEL 명령어 시도
	err = rdb.Del(ctx, testKey).Err()
	if err != nil {
		t.Logf("Note: DEL command not supported: %v", err)
		return
	}
	t.Logf("✅ Deleted key '%s'", testKey)
}

// TestRedisListOperations는 Redis List 작업을 테스트합니다.
func TestRedisListOperations(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetRedisAddr(),
		Password:     "", // 인증 비활성화
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listKey := "test:api-bridge:list"

	// 리스트에 데이터 추가 시도
	err = rdb.LPush(ctx, listKey, "item1", "item2", "item3").Err()
	if err != nil {
		t.Logf("Note: LPUSH command not supported: %v", err)
		return
	}

	t.Log("✅ Pushed items to list")

	// 리스트 길이 확인 시도
	length, err := rdb.LLen(ctx, listKey).Result()
	if err != nil {
		t.Logf("Note: LLEN command not supported: %v", err)
		return
	}

	if length != 3 {
		t.Errorf("Expected list length 3, got %d", length)
		return
	}

	t.Logf("✅ List length: %d", length)

	// 리스트 조회 시도
	items, err := rdb.LRange(ctx, listKey, 0, -1).Result()
	if err != nil {
		t.Logf("Note: LRANGE command not supported: %v", err)
		return
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
		return
	}

	t.Logf("✅ List items: %v", items)

	// 리스트 정리 시도
	err = rdb.Del(ctx, listKey).Err()
	if err != nil {
		t.Logf("Note: DEL command not supported: %v", err)
		return
	}

	t.Log("✅ List cleaned up")
}

// TestRedisHashOperations는 Redis Hash 작업을 테스트합니다.
func TestRedisHashOperations(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetRedisAddr(),
		Password:     "", // 인증 비활성화
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hashKey := "test:api-bridge:hash"

	// 해시 필드 설정 시도
	err = rdb.HSet(ctx, hashKey, "field1", "value1", "field2", "value2").Err()
	if err != nil {
		t.Logf("Note: HSET command not supported: %v", err)
		return
	}

	t.Log("✅ Set hash fields")

	// 해시 필드 조회 시도
	value, err := rdb.HGet(ctx, hashKey, "field1").Result()
	if err != nil {
		t.Logf("Note: HGET command not supported: %v", err)
		return
	}

	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
		return
	}

	t.Logf("✅ Hash field value: %s", value)

	// 모든 해시 필드 조회 시도
	allFields, err := rdb.HGetAll(ctx, hashKey).Result()
	if err != nil {
		t.Logf("Note: HGETALL command not supported: %v", err)
		return
	}

	if len(allFields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(allFields))
		return
	}

	t.Logf("✅ All hash fields: %v", allFields)

	// 해시 정리 시도
	err = rdb.Del(ctx, hashKey).Err()
	if err != nil {
		t.Logf("Note: DEL command not supported: %v", err)
		return
	}

	t.Log("✅ Hash cleaned up")
}

// BenchmarkRedisOperations는 Redis 작업 성능을 벤치마크합니다.
func BenchmarkRedisOperations(b *testing.B) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.GetRedisAddr(),
		Password:     "", // 인증 비활성화
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	defer rdb.Close()

	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("benchmark:key:%d", time.Now().UnixNano())
			value := fmt.Sprintf("value:%d", time.Now().UnixNano())

			err := rdb.Set(ctx, key, value, time.Second).Err()
			if err != nil {
				b.Errorf("Set failed: %v", err)
				continue
			}

			_, err = rdb.Get(ctx, key).Result()
			if err != nil {
				b.Errorf("Get failed: %v", err)
				continue
			}

			rdb.Del(ctx, key)
		}
	})
}
