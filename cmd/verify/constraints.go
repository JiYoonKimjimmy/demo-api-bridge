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

	fmt.Printf("📊 Constraint Verification Tool\n")
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

	// Foreign Key 검증
	fmt.Println("📌 Checking Foreign Key Constraints...")
	fkExpected := map[string]ForeignKeyInfo{
		"FK_ORC_ROUTING": {
			TableName:       "ORCHESTRATION_RULES",
			ColumnName:      "ROUTING_RULE_ID",
			RefTable:        "ROUTING_RULES",
			DeleteRule:      "CASCADE",
		},
		"FK_CMP_ROUTING": {
			TableName:       "COMPARISON_LOGS",
			ColumnName:      "ROUTING_RULE_ID",
			RefTable:        "ROUTING_RULES",
			DeleteRule:      "CASCADE",
		},
	}

	fmt.Println("+------------------+--------------------------+----------+-------------+")
	fmt.Println("| Constraint Name  | Table Name               | Exists   | Delete Rule |")
	fmt.Println("+------------------+--------------------------+----------+-------------+")

	allFKValid := true
	for constraintName, expected := range fkExpected {
		exists, deleteRule := checkForeignKey(db, constraintName, expected.TableName)
		existsStr := "NO"
		if exists {
			existsStr = "YES"
			if strings.ToUpper(deleteRule) != expected.DeleteRule {
				allFKValid = false
			}
		} else {
			allFKValid = false
		}
		fmt.Printf("| %-16s | %-24s | %-8s | %-11s |\n",
			constraintName, expected.TableName, existsStr, deleteRule)
	}
	fmt.Println("+------------------+--------------------------+----------+-------------+")
	fmt.Println()

	// Check Constraint 검증
	fmt.Println("📌 Checking Check Constraints...")
	checkConstraints := []string{
		"CHK_METHOD",
		"CHK_STRATEGY",
		"CHK_IS_ACTIVE",
		"CHK_EP_METHOD",
		"CHK_EP_IS_ACTIVE",
		"CHK_EXEC_TYPE",
		"CHK_ORC_IS_ACTIVE",
		"CHK_CMP_IS_MATCHED",
	}

	fmt.Println("+------------------+--------------------------+--------+")
	fmt.Println("| Constraint Name  | Table Name               | Exists |")
	fmt.Println("+------------------+--------------------------+--------+")

	allCheckValid := true
	for _, constraintName := range checkConstraints {
		exists, tableName := checkCheckConstraint(db, constraintName)
		existsStr := "NO"
		if exists {
			existsStr = "YES"
		} else {
			allCheckValid = false
		}
		fmt.Printf("| %-16s | %-24s | %-6s |\n",
			constraintName, tableName, existsStr)
	}
	fmt.Println("+------------------+--------------------------+--------+")
	fmt.Println()

	if allFKValid && allCheckValid {
		fmt.Println("✅ All constraints exist and are valid!")
		os.Exit(0)
	} else {
		fmt.Println("❌ Some constraints are missing or invalid!")
		os.Exit(1)
	}
}

type ForeignKeyInfo struct {
	TableName  string
	ColumnName string
	RefTable   string
	DeleteRule string
}

func checkForeignKey(db *sql.DB, constraintName, tableName string) (bool, string) {
	var deleteRule string
	err := db.QueryRow(`
		SELECT delete_rule
		FROM user_constraints
		WHERE constraint_name = :1
		  AND table_name = :2
		  AND constraint_type = 'R'
	`, strings.ToUpper(constraintName), strings.ToUpper(tableName)).Scan(&deleteRule)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, ""
		}
		return false, "ERROR"
	}

	return true, deleteRule
}

func checkCheckConstraint(db *sql.DB, constraintName string) (bool, string) {
	var tableName string
	err := db.QueryRow(`
		SELECT table_name
		FROM user_constraints
		WHERE constraint_name = :1
		  AND constraint_type = 'C'
		  AND constraint_name NOT LIKE 'SYS_%'
	`, strings.ToUpper(constraintName)).Scan(&tableName)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, ""
		}
		return false, "ERROR"
	}

	return true, tableName
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
