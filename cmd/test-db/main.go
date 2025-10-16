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
)

func main() {
	fmt.Println("🔍 OracleDB Connection Test")
	fmt.Println("==========================")

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

	// 연결 정보 출력 (비밀번호 마스킹)
	fmt.Printf("🏠 Host: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
	fmt.Printf("🗃️  SID: %s\n", cfg.Database.SID)
	fmt.Printf("👤 Username: %s\n", cfg.Database.Username)
	fmt.Printf("🔐 Password: %s\n", maskPassword(cfg.Database.Password))

	// DSN 생성
	dsn := cfg.Database.GetDSN()
	fmt.Printf("🔗 DSN: %s\n", maskPassword(dsn))

	// 데이터베이스 연결
	fmt.Println("\n🚀 Connecting to OracleDB...")
	db, err := sql.Open("godror", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}
	defer db.Close()

	// 연결 풀 설정
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectionTimeout)
	defer cancel()

	fmt.Println("⏱️  Testing connection...")
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connection successful!")

	// 기본 정보 조회
	fmt.Println("\n📊 Database Information:")
	testBasicQueries(db)

	// 연결 풀 상태
	fmt.Println("\n🔗 Connection Pool Status:")
	printPoolStats(db)

	fmt.Println("\n🎉 All tests passed!")
}

func testBasicQueries(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 현재 시간
	var currentTime time.Time
	err := db.QueryRowContext(ctx, "SELECT SYSDATE FROM DUAL").Scan(&currentTime)
	if err != nil {
		fmt.Printf("❌ Failed to query SYSDATE: %v\n", err)
	} else {
		fmt.Printf("⏰ Current time: %v\n", currentTime)
	}

	// 버전 정보
	var version string
	err = db.QueryRowContext(ctx, "SELECT BANNER FROM V$VERSION WHERE ROWNUM = 1").Scan(&version)
	if err != nil {
		fmt.Printf("❌ Failed to query version: %v\n", err)
	} else {
		fmt.Printf("📦 Version: %s\n", version)
	}

	// 사용자 정보
	var username string
	err = db.QueryRowContext(ctx, "SELECT USER FROM DUAL").Scan(&username)
	if err != nil {
		fmt.Printf("❌ Failed to query user: %v\n", err)
	} else {
		fmt.Printf("👤 Connected as: %s\n", username)
	}

	// 인스턴스 이름
	var instanceName string
	err = db.QueryRowContext(ctx, "SELECT INSTANCE_NAME FROM V$INSTANCE").Scan(&instanceName)
	if err != nil {
		fmt.Printf("❌ Failed to query instance: %v\n", err)
	} else {
		fmt.Printf("🏷️  Instance: %s\n", instanceName)
	}

	// 데이터베이스 이름
	var dbName string
	err = db.QueryRowContext(ctx, "SELECT NAME FROM V$DATABASE").Scan(&dbName)
	if err != nil {
		fmt.Printf("❌ Failed to query database name: %v\n", err)
	} else {
		fmt.Printf("🗄️  Database: %s\n", dbName)
	}
}

func printPoolStats(db *sql.DB) {
	stats := db.Stats()
	fmt.Printf("📈 Open Connections: %d\n", stats.OpenConnections)
	fmt.Printf("🔄 In Use: %d\n", stats.InUse)
	fmt.Printf("😴 Idle: %d\n", stats.Idle)
	fmt.Printf("⏳ Wait Count: %d\n", stats.WaitCount)
	fmt.Printf("⏱️  Wait Duration: %v\n", stats.WaitDuration)
	fmt.Printf("🔒 Max Idle Closed: %d\n", stats.MaxIdleClosed)
	fmt.Printf("🔒 Max Idle Time Closed: %d\n", stats.MaxIdleTimeClosed)
	fmt.Printf("🔒 Max Lifetime Closed: %d\n", stats.MaxLifetimeClosed)
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
