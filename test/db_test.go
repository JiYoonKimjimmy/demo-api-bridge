package test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"demo-api-bridge/pkg/config"

	_ "github.com/sijms/go-ora/v2"
)

// TestOracleConnection은 OracleDB 연결을 테스트합니다.
func TestOracleConnection(t *testing.T) {
	// 설정 로드
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// DSN 생성
	dsn := cfg.Database.GetDSN()
	t.Logf("Connecting to OracleDB with DSN: %s", maskPassword(dsn))
	t.Logf("Actual DSN (for debugging): %s", dsn)

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 연결 설정
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectionTimeout)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Log("✅ OracleDB connection successful!")

	// 기본 쿼리 테스트
	testBasicQuery(t, db)
}

// testBasicQuery는 기본 쿼리를 테스트합니다.
func testBasicQuery(t *testing.T, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 현재 시간 조회
	var currentTime time.Time
	err := db.QueryRowContext(ctx, "SELECT SYSDATE FROM DUAL").Scan(&currentTime)
	if err != nil {
		t.Errorf("Failed to query SYSDATE: %v", err)
		return
	}

	t.Logf("✅ Current Oracle time: %v", currentTime)

	// 버전 정보 조회
	var version string
	err = db.QueryRowContext(ctx, "SELECT BANNER FROM V$VERSION WHERE ROWNUM = 1").Scan(&version)
	if err != nil {
		t.Errorf("Failed to query version: %v", err)
		return
	}

	t.Logf("✅ Oracle version: %s", version)

	// 사용자 정보 조회
	var username string
	err = db.QueryRowContext(ctx, "SELECT USER FROM DUAL").Scan(&username)
	if err != nil {
		t.Errorf("Failed to query user: %v", err)
		return
	}

	t.Logf("✅ Connected as user: %s", username)
}

// TestDatabaseConnectionPool은 연결 풀을 테스트합니다.
func TestDatabaseConnectionPool(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("oracle", cfg.Database.GetDSN())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 연결 풀 설정
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 연결 풀 상태 확인
	stats := db.Stats()
	t.Logf("Database connection pool stats:")
	t.Logf("  OpenConnections: %d", stats.OpenConnections)
	t.Logf("  InUse: %d", stats.InUse)
	t.Logf("  Idle: %d", stats.Idle)
	t.Logf("  WaitCount: %d", stats.WaitCount)
	t.Logf("  WaitDuration: %v", stats.WaitDuration)

	// 다중 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 동시에 여러 쿼리 실행
	for i := 0; i < 5; i++ {
		go func(id int) {
			var result string
			err := db.QueryRowContext(ctx, "SELECT 'Connection test' FROM DUAL").Scan(&result)
			if err != nil {
				t.Errorf("Query %d failed: %v", id, err)
			} else {
				t.Logf("✅ Query %d result: %s", id, result)
			}
		}(i)
	}

	// 잠시 대기
	time.Sleep(2 * time.Second)

	// 최종 연결 풀 상태
	finalStats := db.Stats()
	t.Logf("Final connection pool stats:")
	t.Logf("  OpenConnections: %d", finalStats.OpenConnections)
	t.Logf("  InUse: %d", finalStats.InUse)
	t.Logf("  Idle: %d", finalStats.Idle)
}

// BenchmarkDatabaseConnection은 연결 성능을 벤치마크합니다.
func BenchmarkDatabaseConnection(b *testing.B) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("oracle", cfg.Database.GetDSN())
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 연결 풀 설정
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var result string
			err := db.QueryRowContext(ctx, "SELECT 'benchmark' FROM DUAL").Scan(&result)
			cancel()
			if err != nil {
				b.Errorf("Query failed: %v", err)
			}
		}
	})
}

// maskPassword는 DSN에서 비밀번호를 마스킹합니다.
func maskPassword(dsn string) string {
	// 간단한 마스킹 - 실제로는 더 정교한 처리가 필요할 수 있습니다
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-10:]
	}
	return "***"
}

// TestDatabaseTransaction은 트랜잭션을 테스트합니다.
func TestDatabaseTransaction(t *testing.T) {
	cfg, err := config.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("oracle", cfg.Database.GetDSN())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 트랜잭션 시작
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// 테스트용 임시 테이블 생성 (실제로는 권한이 필요할 수 있음)
	_, err = tx.ExecContext(ctx, "CREATE GLOBAL TEMPORARY TABLE test_temp (id NUMBER, name VARCHAR2(100)) ON COMMIT DELETE ROWS")
	if err != nil {
		t.Logf("Note: Cannot create temporary table (may need privileges): %v", err)
		// 트랜잭션 롤백
		tx.Rollback()
		return
	}

	// 데이터 삽입
	_, err = tx.ExecContext(ctx, "INSERT INTO test_temp (id, name) VALUES (1, 'test')")
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
		tx.Rollback()
		return
	}

	// 데이터 조회
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_temp").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query data: %v", err)
		tx.Rollback()
		return
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// 트랜잭션 커밋
	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}

	t.Log("✅ Database transaction test successful!")
}
