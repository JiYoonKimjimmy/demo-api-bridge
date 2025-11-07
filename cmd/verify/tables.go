package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/sijms/go-ora/v2"
)

func main() {
	var env = flag.String("env", "development", "Environment (development, staging, production)")
	flag.Parse()

	fmt.Printf("📊 Table Verification Tool\n")
	fmt.Printf("   Environment: %s\n\n", *env)

	// 환경별 DSN 설정
	dsn := getDSNByEnv(*env)
	if dsn == "" {
		fmt.Printf("❌ Unknown environment: %s\n", *env)
		os.Exit(1)
	}

	// 데이터베이스 연결
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Connected to database (%s)\n\n", *env)

	// 검증할 테이블 목록
	tables := []string{
		"ROUTING_RULES",
		"API_ENDPOINTS",
		"ORCHESTRATION_RULES",
		"COMPARISON_LOGS",
		"GORP_MIGRATIONS",
	}

	fmt.Println("Checking tables...")
	fmt.Println("+-------------------------+--------+-----------+")
	fmt.Println("| Table Name              | Exists | Row Count |")
	fmt.Println("+-------------------------+--------+-----------+")

	allExist := true
	for _, tableName := range tables {
		exists, rowCount := checkTable(db, tableName)
		status := "NO"
		if exists {
			status = "YES"
		} else {
			allExist = false
		}
		fmt.Printf("| %-23s | %-6s | %-9d |\n", tableName, status, rowCount)
	}

	fmt.Println("+-------------------------+--------+-----------+")
	fmt.Println()

	if allExist {
		fmt.Println("✅ All tables exist!")
		os.Exit(0)
	} else {
		fmt.Println("❌ Some tables are missing!")
		os.Exit(1)
	}
}

func checkTable(db *sql.DB, tableName string) (bool, int) {
	// 테이블 존재 여부 확인
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_tables
		WHERE table_name = :1
	`, tableName).Scan(&count)

	if err != nil || count == 0 {
		return false, 0
	}

	// 행 개수 조회
	var rowCount int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err = db.QueryRow(query).Scan(&rowCount)
	if err != nil {
		return true, 0
	}

	return true, rowCount
}

func getDSNByEnv(env string) string {
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		fmt.Println("📌 Using DATABASE_DSN from environment variable")
		return dsn
	}

	switch env {
	case "development":
		return "oracle://map:StgMAP1104%23@dev1-db.konadc.com:15322/kmdbp19"
	case "staging":
		return "oracle://map:StgMAP1104%23@dev3-db.konadc.com:15321/kmdbp"
	case "production":
		return "oracle://map:StgMAP1104%23@db.konadc.com:15321/kmdbp"
	default:
		return ""
	}
}
