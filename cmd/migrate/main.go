package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

	// 마이그레이션 실행 또는 상태 확인
	var n int
	if *direction == "status" {
		// 상태 확인
		if err := showMigrationStatus(db, migrations); err != nil {
			fmt.Printf("❌ Failed to get migration status: %v\n", err)
			os.Exit(1)
		}
		return
	} else if *direction == "up" {
		fmt.Println("🚀 Applying migrations...")
		n, err = migrate.ExecMax(db, "oci8", migrations, migrate.Up, *limit)
	} else if *direction == "down" {
		fmt.Println("⏪ Rolling back migrations...")
		n, err = migrate.ExecMax(db, "oci8", migrations, migrate.Down, *limit)
	} else {
		fmt.Printf("❌ Invalid direction: %s (use 'up', 'down', or 'status')\n", *direction)
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

	// 기본값 (dbconfig.yml과 동일한 환경 정보) - sijms/go-ora 형식
	switch env {
	case "development":
		// dbconfig.yml development 환경과 동일
		return "oracle://map:StgMAP1104%23@dev1-db.konadc.com:15322/kmdbp19"
	case "staging":
		// dbconfig.yml staging 환경과 동일
		return "oracle://map:StgMAP1104%23@dev3-db.konadc.com:15321/kmdbp"
	case "production":
		// dbconfig.yml production 환경과 동일
		// 프로덕션은 반드시 환경 변수 사용 권장
		fmt.Println("⚠️  WARNING: Using hardcoded production credentials!")
		fmt.Println("   Recommended: export DATABASE_DSN='oracle://user:pass@host:port/sid'")
		return "oracle://map:StgMAP1104%23@db.konadc.com:15321/kmdbp"
	default:
		return ""
	}
}

// showMigrationStatus는 현재 마이그레이션 상태를 보여줍니다
func showMigrationStatus(db *sql.DB, source *migrate.FileMigrationSource) error {
	fmt.Println("📊 Migration Status")
	fmt.Println()

	// 적용된 마이그레이션 기록 조회
	records, err := migrate.GetMigrationRecords(db, "oci8")
	if err != nil {
		return fmt.Errorf("failed to get migration records: %w", err)
	}

	// 적용된 마이그레이션을 맵으로 저장
	applied := make(map[string]bool)
	for _, record := range records {
		applied[record.Id] = true
	}

	// 모든 마이그레이션 파일 조회
	migrations, err := source.FindMigrations()
	if err != nil {
		return fmt.Errorf("failed to find migrations: %w", err)
	}

	// ID로 정렬
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Id < migrations[j].Id
	})

	// 테이블 헤더 출력
	fmt.Println("+----+--------------------------------------------------+---------+")
	fmt.Println("| ID | Migration                                        | Applied |")
	fmt.Println("+----+--------------------------------------------------+---------+")

	// 각 마이그레이션 상태 출력
	appliedCount := 0
	for _, m := range migrations {
		status := "no"
		if applied[m.Id] {
			status = "yes"
			appliedCount++
		}
		filename := filepath.Base(m.Id)
		// 파일명이 길면 자르기
		if len(filename) > 48 {
			filename = filename[:45] + "..."
		}
		fmt.Printf("| %-2s | %-48s | %-7s |\n", m.Id, filename, status)
	}

	fmt.Println("+----+--------------------------------------------------+---------+")
	fmt.Printf("\n✅ Applied: %d/%d migrations\n", appliedCount, len(migrations))

	return nil
}
