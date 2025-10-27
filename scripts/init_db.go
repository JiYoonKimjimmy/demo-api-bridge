package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

func main() {
	// Oracle DB 연결 정보 (config.yaml 기반)
	host := "dev1-db.konadc.com"
	port := 15322
	sid := "kmdbp19"
	username := "map"
	password := "StgMAP1104#"

	// DSN 생성 (패스워드 URL 인코딩)
	encodedPassword := url.QueryEscape(password)
	dsn := fmt.Sprintf("oracle://%s:%s@%s:%d/%s", username, encodedPassword, host, port, sid)

	// 데이터베이스 연결
	log.Println("Connecting to Oracle database...")
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database successfully")

	// SQL 파일 읽기
	sqlFile := "scripts/create_tables.sql"
	log.Printf("Reading SQL file: %s", sqlFile)

	content, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("Failed to read SQL file: %v", err)
	}

	// SQL 문을 파싱
	statements := parseSQL(string(content))

	// 각 SQL 문 실행
	successCount := 0
	skipCount := 0
	errorCount := 0

	for i, stmt := range statements {
		log.Printf("Executing statement %d...", i+1)
		log.Printf("Statement preview: %s...", stmt[:min(80, len(stmt))])

		_, err := db.ExecContext(ctx, stmt)
		if err != nil {
			// 이미 존재하는 테이블 에러는 무시
			errMsg := err.Error()
			if strings.Contains(errMsg, "ORA-00955") { // name is already used
				log.Printf("⚠️  Statement %d already exists, skipped", i+1)
				skipCount++
				continue
			}

			log.Printf("❌ Error executing statement %d: %v", i+1, err)
			errorCount++
			continue
		}

		successCount++
		log.Printf("✅ Statement %d executed successfully", i+1)
	}

	// 결과 출력
	log.Println("\n" + strings.Repeat("=", 50))
	log.Printf("✅ Successfully executed: %d statements", successCount)
	log.Printf("⚠️  Skipped: %d statements", skipCount)
	log.Printf("❌ Failed: %d statements", errorCount)
	log.Println(strings.Repeat("=", 50))

	if errorCount > 0 {
		log.Println("\n⚠️  Some statements failed. Please check the errors above.")
		os.Exit(1)
	}

	log.Println("\n🎉 Database initialization completed successfully!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseSQL은 SQL 파일 내용을 개별 SQL 문으로 파싱합니다
func parseSQL(content string) []string {
	var statements []string
	var currentStmt strings.Builder
	inBlockComment := false
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 블록 주석 시작
		if strings.HasPrefix(trimmedLine, "/*") {
			inBlockComment = true
		}

		// 블록 주석 중이거나 한 줄 주석이면 스킵
		if inBlockComment || strings.HasPrefix(trimmedLine, "--") {
			// 블록 주석 종료 확인
			if strings.HasSuffix(trimmedLine, "*/") {
				inBlockComment = false
			}
			continue
		}

		// 빈 줄이면 스킵
		if trimmedLine == "" {
			continue
		}

		// 현재 문장에 추가
		if currentStmt.Len() > 0 {
			currentStmt.WriteString(" ")
		}
		currentStmt.WriteString(trimmedLine)

		// 세미콜론으로 끝나면 문장 완료
		if strings.HasSuffix(trimmedLine, ";") {
			stmt := strings.TrimSpace(strings.TrimSuffix(currentStmt.String(), ";"))

			// DDL 문인지 확인
			upperStmt := strings.ToUpper(stmt)
			if strings.HasPrefix(upperStmt, "CREATE") ||
				strings.HasPrefix(upperStmt, "ALTER") ||
				strings.HasPrefix(upperStmt, "DROP") ||
				strings.HasPrefix(upperStmt, "COMMENT") {
				statements = append(statements, stmt)
			}

			currentStmt.Reset()
		}
	}

	// 마지막 문장이 세미콜론 없이 끝난 경우
	if currentStmt.Len() > 0 {
		stmt := strings.TrimSpace(currentStmt.String())
		upperStmt := strings.ToUpper(stmt)
		if strings.HasPrefix(upperStmt, "CREATE") ||
			strings.HasPrefix(upperStmt, "ALTER") ||
			strings.HasPrefix(upperStmt, "DROP") ||
			strings.HasPrefix(upperStmt, "COMMENT") {
			statements = append(statements, stmt)
		}
	}

	return statements
}
