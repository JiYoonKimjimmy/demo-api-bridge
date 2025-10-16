package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"demo-api-bridge/pkg/config"

	_ "github.com/godror/godror"
	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("🔍 API Bridge Database Connection Test")
	fmt.Println("=====================================")

	// 설정 파일 경로 확인
	configPath := "config/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 설정 로드
	fmt.Printf("📁 Loading config from: %s\n", configPath)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// OracleDB 테스트
	fmt.Println("\n🗄️  Testing OracleDB Connection...")
	if !testOracleDB(cfg) {
		log.Fatal("❌ OracleDB connection test failed")
	}

	// Redis 테스트
	fmt.Println("\n🔴 Testing Redis Connection...")
	if !testRedis(cfg) {
		log.Fatal("❌ Redis connection test failed")
	}

	// 통합 테스트
	fmt.Println("\n🔗 Testing Integration...")
	if !testIntegration(cfg) {
		log.Fatal("❌ Integration test failed")
	}

	fmt.Println("\n🎉 All database connection tests passed!")
}

func testOracleDB(cfg *config.Config) bool {
	// DSN 생성
	dsn := cfg.Database.GetDSN()
	fmt.Printf("🔗 OracleDB DSN: %s\n", maskPassword(dsn))

	// 데이터베이스 연결
	db, err := sql.Open("godror", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to open OracleDB: %v\n", err)
		return false
	}
	defer db.Close()

	// 연결 풀 설정
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectionTimeout)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to ping OracleDB: %v\n", err)
		return false
	}

	fmt.Println("✅ OracleDB connection successful!")

	// 기본 쿼리 테스트
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var currentTime time.Time
	err = db.QueryRowContext(ctx, "SELECT SYSDATE FROM DUAL").Scan(&currentTime)
	if err != nil {
		fmt.Printf("❌ Failed to query SYSDATE: %v\n", err)
		return false
	}

	fmt.Printf("⏰ Oracle time: %v\n", currentTime)

	// 연결 풀 상태
	stats := db.Stats()
	fmt.Printf("📊 Connection pool - Open: %d, InUse: %d, Idle: %d\n",
		stats.OpenConnections, stats.InUse, stats.Idle)

	return true
}

func testRedis(cfg *config.Config) bool {
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

	fmt.Printf("🔗 Redis address: %s\n", cfg.Redis.GetRedisAddr())

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	if err != nil {
		fmt.Printf("❌ Failed to ping Redis: %v\n", err)
		return false
	}

	fmt.Println("✅ Redis connection successful!")

	// 기본 작업 테스트
	testKey := "test:api-bridge:integration"
	testValue := fmt.Sprintf("test-value-%d", time.Now().Unix())

	// 데이터 저장
	err = rdb.Set(ctx, testKey, testValue, 10*time.Second).Err()
	if err != nil {
		fmt.Printf("❌ Failed to set Redis key: %v\n", err)
		return false
	}

	// 데이터 조회
	value, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		fmt.Printf("❌ Failed to get Redis key: %v\n", err)
		return false
	}

	if value != testValue {
		fmt.Printf("❌ Redis value mismatch: expected '%s', got '%s'\n", testValue, value)
		return false
	}

	fmt.Printf("✅ Redis operation successful: %s = %s\n", testKey, value)

	// 정리
	rdb.Del(ctx, testKey)

	// 연결 풀 상태
	poolStats := rdb.PoolStats()
	fmt.Printf("📊 Redis pool - TotalConns: %d, IdleConns: %d, StaleConns: %d\n",
		poolStats.TotalConns, poolStats.IdleConns, poolStats.StaleConns)

	return true
}

func testIntegration(cfg *config.Config) bool {
	// OracleDB 연결
	oracleDSN := cfg.Database.GetDSN()
	db, err := sql.Open("godror", oracleDSN)
	if err != nil {
		fmt.Printf("❌ Failed to open OracleDB for integration test: %v\n", err)
		return false
	}
	defer db.Close()

	// Redis 연결
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// OracleDB에서 데이터 조회
	var oracleTime time.Time
	err = db.QueryRowContext(ctx, "SELECT SYSDATE FROM DUAL").Scan(&oracleTime)
	if err != nil {
		fmt.Printf("❌ Failed to query OracleDB in integration test: %v\n", err)
		return false
	}

	// Redis에 OracleDB 결과 저장
	cacheKey := "integration:oracle:time"
	cacheValue := oracleTime.Format(time.RFC3339)

	err = rdb.Set(ctx, cacheKey, cacheValue, 60*time.Second).Err()
	if err != nil {
		fmt.Printf("❌ Failed to cache OracleDB result in Redis: %v\n", err)
		return false
	}

	// Redis에서 데이터 조회
	cachedValue, err := rdb.Get(ctx, cacheKey).Result()
	if err != nil {
		fmt.Printf("❌ Failed to get cached value from Redis: %v\n", err)
		return false
	}

	if cachedValue != cacheValue {
		fmt.Printf("❌ Cache value mismatch: expected '%s', got '%s'\n", cacheValue, cachedValue)
		return false
	}

	fmt.Printf("✅ Integration test successful!\n")
	fmt.Printf("📊 OracleDB time: %v\n", oracleTime)
	fmt.Printf("💾 Cached in Redis: %s\n", cachedValue)

	// 정리
	rdb.Del(ctx, cacheKey)

	return true
}

func maskPassword(password string) string {
	if len(password) == 0 {
		return "(empty)"
	}
	if len(password) <= 4 {
		return "***"
	}
	return password[:2] + "***" + password[len(password)-2:]
}
