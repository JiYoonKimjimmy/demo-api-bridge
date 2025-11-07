package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/sijms/go-ora/v2"
)

func main() {
	var env = flag.String("env", "development", "Environment (development, staging, production)")
	flag.Parse()

	fmt.Printf("📊 Index Verification Tool\n")
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

	// 검증할 인덱스 목록 (테이블별)
	expectedIndexes := map[string][]string{
		"ROUTING_RULES": {
			"IDX_ROUTING_PATH",
			"IDX_ROUTING_ENDPOINT",
			"IDX_ROUTING_ACTIVE",
			"IDX_ROUTING_PATH_METHOD", // 복합 인덱스
		},
		"API_ENDPOINTS": {
			"IDX_EP_NAME",
			"IDX_EP_ACTIVE",
			"IDX_EP_URL_METHOD", // 복합 인덱스
		},
		"ORCHESTRATION_RULES": {
			"IDX_ORC_ROUTING",
			"IDX_ORC_NAME",
		},
		"COMPARISON_LOGS": {
			"IDX_CMP_ROUTING",
			"IDX_CMP_CREATED",
			"IDX_CMP_MATCHED",
		},
	}

	fmt.Println("Checking indexes...")
	fmt.Println("+-------------------------+---------------------------+--------+--------+")
	fmt.Println("| Table Name              | Index Name                | Exists | Status |")
	fmt.Println("+-------------------------+---------------------------+--------+--------+")

	allValid := true
	totalExpected := 0
	totalFound := 0

	for tableName, indexes := range expectedIndexes {
		for _, indexName := range indexes {
			totalExpected++
			exists, status := checkIndex(db, tableName, indexName)
			existsStr := "NO"
			if exists {
				existsStr = "YES"
				totalFound++
			} else {
				allValid = false
			}

			if status != "VALID" && status != "" {
				allValid = false
			}

			fmt.Printf("| %-23s | %-25s | %-6s | %-6s |\n",
				tableName, indexName, existsStr, status)
		}
	}

	fmt.Println("+-------------------------+---------------------------+--------+--------+")
	fmt.Printf("\n📊 Summary: %d/%d indexes found\n", totalFound, totalExpected)

	if allValid && totalFound == totalExpected {
		fmt.Println("✅ All indexes exist and are valid!")
		os.Exit(0)
	} else {
		fmt.Println("❌ Some indexes are missing or invalid!")
		os.Exit(1)
	}
}

func checkIndex(db *sql.DB, tableName, indexName string) (bool, string) {
	var status string
	err := db.QueryRow(`
		SELECT status
		FROM user_indexes
		WHERE table_name = :1
		  AND index_name = :2
	`, strings.ToUpper(tableName), strings.ToUpper(indexName)).Scan(&status)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, ""
		}
		return false, "ERROR"
	}

	return true, status
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
