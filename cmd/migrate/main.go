package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/sijms/go-ora/v2"
	migrate "github.com/rubenv/sql-migrate"
)

func main() {
	var (
		env       = flag.String("env", "development", "Environment (development, staging, production)")
		direction = flag.String("direction", "up", "Migration direction (up, down)")
		limit     = flag.Int("limit", 0, "Limit number of migrations (0 = all)")
	)
	flag.Parse()

	fmt.Printf("🔧 Starting migration tool...\n")
	fmt.Printf("   Environment: %s\n", *env)
	fmt.Printf("   Direction: %s\n", *direction)
	if *limit > 0 {
		fmt.Printf("   Limit: %d migration(s)\n", *limit)
	} else {
		fmt.Printf("   Limit: all migrations\n")
	}
	fmt.Println()

	// 환경별 DSN 설정
	dsn := getDSNByEnv(*env)
	if dsn == "" {
		fmt.Printf("❌ Unknown environment: %s\n", *env)
		fmt.Println("   Available environments: development, staging, production")
		os.Exit(1)
	}

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 연결 테스트
	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Failed to ping database: %v\n", err)
		fmt.Println("   Please check your database connection settings")
		os.Exit(1)
	}

	fmt.Printf("✅ Connected to database (%s)\n\n", *env)

	// 마이그레이션 테이블 설정
	migrate.SetTable("gorp_migrations")

	// 마이그레이션 소스 설정
	migrations := &migrate.FileMigrationSource{
		Dir: "db/migrations",
	}

	// 마이그레이션 실행 (oci8 dialect 사용)
	var n int
	if *direction == "up" {
		fmt.Println("🚀 Applying migrations...")
		n, err = migrate.ExecMax(db, "oci8", migrations, migrate.Up, *limit)
	} else if *direction == "down" {
		fmt.Println("⏪ Rolling back migrations...")
		n, err = migrate.ExecMax(db, "oci8", migrations, migrate.Down, *limit)
	} else {
		fmt.Printf("❌ Invalid direction: %s (use 'up' or 'down')\n", *direction)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("❌ Migration failed: %v\n", err)
		os.Exit(1)
	}

	if n > 0 {
		fmt.Printf("✅ Applied %d migration(s) successfully (%s)!\n", n, *direction)
	} else {
		fmt.Println("ℹ️  No migrations to apply (database is up to date)")
	}
}

// getDSNByEnv는 환경별 DSN을 반환합니다 (환경 변수 우선)
func getDSNByEnv(env string) string {
	// 환경 변수에서 먼저 확인 (보안 권장)
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		fmt.Println("📌 Using DATABASE_DSN from environment variable")
		return dsn
	}

	// 기본값 (개발 환경) - sijms/go-ora 형식
	switch env {
	case "development":
		// config.yaml의 database 설정과 동일
		return "oracle://map:StgMAP1104%23@dev1-db.konadc.com:15322/kmdbp19"
	case "staging":
		// Staging 환경 설정 (필요 시 수정)
		return "oracle://map:StgMAP1104%23@staging-host:1521/STAGINGDB"
	case "production":
		// Production 환경 설정 (필요 시 수정)
		// 프로덕션은 반드시 환경 변수 사용 권장
		fmt.Println("⚠️  WARNING: Using hardcoded production credentials!")
		fmt.Println("   Recommended: export DATABASE_DSN='oracle://user:pass@host:port/sid'")
		return "oracle://map:CHANGE_ME@prod-host:1521/PRODDB"
	default:
		return ""
	}
}
