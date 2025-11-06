# DB 마이그레이션 가이드 (sql-migrate)

API Bridge 프로젝트의 OracleDB 스키마 마이그레이션을 위한 sql-migrate 적용 가이드입니다.

---

## 📋 목차

1. [sql-migrate 소개](#sql-migrate-소개)
2. [설치 및 설정](#설치-및-설정)
3. [프로젝트 구조](#프로젝트-구조)
4. [마이그레이션 파일 작성](#마이그레이션-파일-작성)
5. [CLI 도구 사용법](#cli-도구-사용법)
6. [Go 코드에서 실행](#go-코드에서-실행)
7. [베스트 프랙티스](#베스트-프랙티스)
8. [트러블슈팅](#트러블슈팅)

---

## sql-migrate 소개

### 왜 sql-migrate인가?

**sql-migrate**는 Go 프로젝트에서 데이터베이스 스키마 버전 관리를 위한 마이그레이션 도구입니다.

#### 주요 특징

- ✅ **OracleDB 공식 지원** (godror/oci8 드라이버)
- ✅ **양방향 마이그레이션** (Up/Down)
- ✅ **CLI와 Go 라이브러리** 모두 지원
- ✅ **트랜잭션 기반** 마이그레이션
- ✅ **임베디드 마이그레이션** 지원 (go:embed)

#### 다른 도구와의 비교

| 기능 | sql-migrate | golang-migrate | goose | Atlas |
|------|-------------|----------------|-------|-------|
| Oracle 지원 | ✅ 공식 | ⚠️ 제한적 | ❌ | 💰 유료 |
| CLI 도구 | ✅ | ✅ | ✅ | ✅ |
| Go 라이브러리 | ✅ | ✅ | ✅ | ✅ |
| 양방향 마이그레이션 | ✅ | ✅ | ✅ | ⚠️ |
| 커뮤니티 | 1,877+ | 12k+ | 5k+ | 4k+ |

---

## 설치 및 설정

### 1. sql-migrate 설치

#### Go 모듈에 추가

```bash
# sql-migrate 라이브러리 설치
go get -tags oracle github.com/rubenv/sql-migrate

# godror 드라이버 (프로젝트에 이미 포함됨)
# go get github.com/godror/godror
```

#### CLI 도구 설치 (선택사항)

```bash
# CLI 도구를 전역으로 설치
go install -tags oracle github.com/rubenv/sql-migrate/...@latest

# 설치 확인
sql-migrate --version
```

**Windows 환경 참고**:
- `GOPATH/bin` 디렉토리가 PATH에 등록되어 있어야 합니다
- 기본 경로: `C:\Users\<사용자명>\go\bin`

### 2. dbconfig.yml 설정

프로젝트 루트에 `dbconfig.yml` 파일을 생성합니다:

```yaml
# dbconfig.yml
development:
  dialect: oracle
  datasource: "user=\"DEMO_USER\" password=\"demo_password\" connectString=\"localhost:1521/XEPDB1\""
  dir: db/migrations
  table: schema_migrations

staging:
  dialect: oracle
  datasource: "user=\"DEMO_USER\" password=\"demo_password\" connectString=\"staging-host:1521/STAGINGDB\""
  dir: db/migrations
  table: schema_migrations

production:
  dialect: oracle
  datasource: "user=\"DEMO_USER\" password=\"demo_password\" connectString=\"prod-host:1521/PRODDB\""
  dir: db/migrations
  table: schema_migrations
```

**보안 권장사항**:
- `.gitignore`에 `dbconfig.yml` 추가
- 프로덕션 환경에서는 환경 변수 사용 (아래 예제 참조)

---

## 프로젝트 구조

### 권장 디렉토리 구조

```
demo-api-bridge/
├── cmd/
│   ├── api-bridge/
│   │   └── main.go
│   └── migrate/              # 마이그레이션 CLI 도구
│       └── main.go           # 마이그레이션 전용 실행 파일
├── db/
│   └── migrations/           # 마이그레이션 파일 디렉토리
│       ├── 20250105_001_create_routing_rules.sql
│       ├── 20250105_002_create_api_endpoints.sql
│       ├── 20250105_003_create_orchestration_rules.sql
│       └── 20250105_004_add_indexes.sql
├── dbconfig.yml              # 마이그레이션 설정 (gitignore 대상)
├── dbconfig.example.yml      # 설정 템플릿 (Git에 포함)
└── docs/
    └── DB_MIGRATION_GUIDE.md # 이 문서
```

### 디렉토리 생성

```bash
# Windows PowerShell
mkdir -p db/migrations
mkdir -p cmd/migrate

# Git Bash / Linux
mkdir -p db/migrations cmd/migrate
```

---

## 마이그레이션 파일 작성

### 파일 명명 규칙

```
<timestamp>_<sequence>_<description>.sql
```

**예시**:
- `20250105_001_create_routing_rules.sql`
- `20250105_002_create_api_endpoints.sql`
- `20250106_003_add_user_index.sql`

### 마이그레이션 파일 구조

각 `.sql` 파일은 **Up**과 **Down** 섹션으로 구성됩니다:

```sql
-- +migrate Up
-- 스키마 변경 (적용)

-- +migrate Down
-- 스키마 롤백 (되돌리기)
```

### 실제 예제

#### 1. 라우팅 규칙 테이블 생성

**파일명**: `db/migrations/20250105_001_create_routing_rules.sql`

```sql
-- +migrate Up
-- RoutingRule 테이블 생성
CREATE TABLE routing_rules (
    id VARCHAR2(36) PRIMARY KEY,
    endpoint_id VARCHAR2(36) NOT NULL,
    request_path VARCHAR2(500) NOT NULL,
    method VARCHAR2(10) NOT NULL,
    strategy VARCHAR2(50) NOT NULL,
    priority NUMBER(10) DEFAULT 0,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_method CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    CONSTRAINT chk_strategy CHECK (strategy IN ('direct', 'orchestration', 'comparison', 'ab_test')),
    CONSTRAINT chk_is_active CHECK (is_active IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_routing_path ON routing_rules(request_path);
CREATE INDEX idx_routing_endpoint ON routing_rules(endpoint_id);
CREATE INDEX idx_routing_active ON routing_rules(is_active);

-- 코멘트 추가
COMMENT ON TABLE routing_rules IS 'API 라우팅 규칙 관리';
COMMENT ON COLUMN routing_rules.strategy IS 'direct: 단일 전달, orchestration: 오케스트레이션, comparison: AB 비교';

-- +migrate Down
-- 테이블 삭제 (롤백)
DROP TABLE routing_rules CASCADE CONSTRAINTS;
```

#### 2. API 엔드포인트 테이블 생성

**파일명**: `db/migrations/20250105_002_create_api_endpoints.sql`

```sql
-- +migrate Up
-- APIEndpoint 테이블 생성
CREATE TABLE api_endpoints (
    id VARCHAR2(36) PRIMARY KEY,
    name VARCHAR2(100) NOT NULL,
    base_url VARCHAR2(500) NOT NULL,
    path VARCHAR2(500),
    method VARCHAR2(10) NOT NULL,
    timeout_ms NUMBER(10) DEFAULT 5000,
    retry_count NUMBER(3) DEFAULT 3,
    headers CLOB,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ep_method CHECK (method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')),
    CONSTRAINT chk_ep_is_active CHECK (is_active IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_ep_name ON api_endpoints(name);
CREATE INDEX idx_ep_active ON api_endpoints(is_active);

-- 코멘트 추가
COMMENT ON TABLE api_endpoints IS '외부 API 엔드포인트 정보';
COMMENT ON COLUMN api_endpoints.headers IS 'JSON 형식의 HTTP 헤더';

-- +migrate Down
DROP TABLE api_endpoints CASCADE CONSTRAINTS;
```

#### 3. 오케스트레이션 규칙 테이블 생성

**파일명**: `db/migrations/20250105_003_create_orchestration_rules.sql`

```sql
-- +migrate Up
-- OrchestrationRule 테이블 생성
CREATE TABLE orchestration_rules (
    id VARCHAR2(36) PRIMARY KEY,
    routing_rule_id VARCHAR2(36) NOT NULL,
    name VARCHAR2(100) NOT NULL,
    execution_type VARCHAR2(20) DEFAULT 'sequential',
    steps CLOB NOT NULL,
    is_active NUMBER(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_orc_routing FOREIGN KEY (routing_rule_id) REFERENCES routing_rules(id) ON DELETE CASCADE,
    CONSTRAINT chk_exec_type CHECK (execution_type IN ('sequential', 'parallel')),
    CONSTRAINT chk_orc_is_active CHECK (is_active IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_orc_routing ON orchestration_rules(routing_rule_id);
CREATE INDEX idx_orc_name ON orchestration_rules(name);

-- 코멘트 추가
COMMENT ON TABLE orchestration_rules IS 'API 오케스트레이션 규칙';
COMMENT ON COLUMN orchestration_rules.steps IS 'JSON 배열 형식의 실행 스텝';

-- +migrate Down
DROP TABLE orchestration_rules CASCADE CONSTRAINTS;
```

#### 4. 비교 로그 테이블 생성

**파일명**: `db/migrations/20250105_004_create_comparison_logs.sql`

```sql
-- +migrate Up
-- ComparisonLog 테이블 생성
CREATE TABLE comparison_logs (
    id VARCHAR2(36) PRIMARY KEY,
    routing_rule_id VARCHAR2(36) NOT NULL,
    request_id VARCHAR2(100),
    old_response CLOB,
    new_response CLOB,
    is_matched NUMBER(1) DEFAULT 0,
    difference_details CLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cmp_routing FOREIGN KEY (routing_rule_id) REFERENCES routing_rules(id) ON DELETE CASCADE,
    CONSTRAINT chk_cmp_is_matched CHECK (is_matched IN (0, 1))
);

-- 인덱스 생성
CREATE INDEX idx_cmp_routing ON comparison_logs(routing_rule_id);
CREATE INDEX idx_cmp_created ON comparison_logs(created_at);
CREATE INDEX idx_cmp_matched ON comparison_logs(is_matched);

-- 코멘트 추가
COMMENT ON TABLE comparison_logs IS 'API 응답 비교 로그';

-- Partitioning by created_at (선택사항 - 대용량 로그 관리)
-- ALTER TABLE comparison_logs PARTITION BY RANGE (created_at) INTERVAL (NUMTODSINTERVAL(30, 'DAY'))
-- (PARTITION p_initial VALUES LESS THAN (TO_DATE('2025-01-01', 'YYYY-MM-DD')));

-- +migrate Down
DROP TABLE comparison_logs CASCADE CONSTRAINTS;
```

#### 5. 복합 인덱스 및 성능 최적화

**파일명**: `db/migrations/20250105_005_add_performance_indexes.sql`

```sql
-- +migrate Up
-- 복합 인덱스 추가 (성능 최적화)
CREATE INDEX idx_routing_path_method ON routing_rules(request_path, method, is_active);
CREATE INDEX idx_ep_url_method ON api_endpoints(base_url, method, is_active);

-- 통계 정보 수집 (Oracle Optimizer)
EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'ROUTING_RULES');
EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'API_ENDPOINTS');
EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'ORCHESTRATION_RULES');
EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'COMPARISON_LOGS');

-- +migrate Down
DROP INDEX idx_routing_path_method;
DROP INDEX idx_ep_url_method;
```

---

## CLI 도구 사용법

### 기본 명령어

#### 1. 마이그레이션 상태 확인

```bash
# 현재 마이그레이션 상태 조회
sql-migrate status -config=dbconfig.yml -env=development
```

**출력 예시**:
```
+-------------------------------+---------+
|          MIGRATION            | APPLIED |
+-------------------------------+---------+
| 20250105_001_create_routing   | yes     |
| 20250105_002_create_endpoints | yes     |
| 20250105_003_create_orchestr  | no      |
+-------------------------------+---------+
```

#### 2. 마이그레이션 적용 (Up)

```bash
# 모든 마이그레이션 적용
sql-migrate up -config=dbconfig.yml -env=development

# 특정 개수만 적용
sql-migrate up -limit=1 -config=dbconfig.yml -env=development
```

#### 3. 마이그레이션 롤백 (Down)

```bash
# 마지막 마이그레이션 1개 롤백
sql-migrate down -limit=1 -config=dbconfig.yml -env=development

# 모든 마이그레이션 롤백 (주의!)
sql-migrate down -config=dbconfig.yml -env=development
```

#### 4. 특정 버전으로 이동

```bash
# Redo: 마지막 마이그레이션을 롤백 후 다시 적용
sql-migrate redo -config=dbconfig.yml -env=development
```

### 환경별 실행

```bash
# Development
sql-migrate up -config=dbconfig.yml -env=development

# Staging
sql-migrate up -config=dbconfig.yml -env=staging

# Production (신중하게!)
sql-migrate up -config=dbconfig.yml -env=production
```

---

## Go 코드에서 실행

### 1. 마이그레이션 전용 CLI 도구 작성

**파일**: `cmd/migrate/main.go`

```go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/godror/godror"
	migrate "github.com/rubenv/sql-migrate"
)

func main() {
	var (
		env       = flag.String("env", "development", "Environment (development, staging, production)")
		direction = flag.String("direction", "up", "Migration direction (up, down)")
		limit     = flag.Int("limit", 0, "Limit number of migrations (0 = all)")
	)
	flag.Parse()

	// 환경별 DSN 설정
	dsn := getDSNByEnv(*env)
	if dsn == "" {
		fmt.Printf("❌ Unknown environment: %s\n", *env)
		os.Exit(1)
	}

	// 데이터베이스 연결
	db, err := sql.Open("godror", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 연결 테스트
	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Connected to database (%s)\n", *env)

	// 마이그레이션 소스 설정
	migrations := &migrate.FileMigrationSource{
		Dir: "db/migrations",
	}

	// 마이그레이션 실행
	var n int
	if *direction == "up" {
		n, err = migrate.ExecMax(db, "oracle", migrations, migrate.Up, *limit)
	} else if *direction == "down" {
		n, err = migrate.ExecMax(db, "oracle", migrations, migrate.Down, *limit)
	} else {
		fmt.Printf("❌ Invalid direction: %s (use 'up' or 'down')\n", *direction)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("❌ Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Applied %d migration(s) (%s)!\n", n, *direction)
}

// getDSNByEnv는 환경별 DSN을 반환합니다 (환경 변수 우선)
func getDSNByEnv(env string) string {
	// 환경 변수에서 먼저 확인 (보안 권장)
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}

	// 기본값 (개발 환경)
	switch env {
	case "development":
		return `user="DEMO_USER" password="demo_password" connectString="localhost:1521/XEPDB1"`
	case "staging":
		return `user="DEMO_USER" password="demo_password" connectString="staging-host:1521/STAGINGDB"`
	case "production":
		return `user="DEMO_USER" password="demo_password" connectString="prod-host:1521/PRODDB"`
	default:
		return ""
	}
}
```

### 2. 실행 방법

```bash
# 개발 환경에 모든 마이그레이션 적용
go run cmd/migrate/main.go -env=development -direction=up

# 마지막 1개 마이그레이션 롤백
go run cmd/migrate/main.go -env=development -direction=down -limit=1

# 프로덕션 환경 (환경 변수 사용)
export DATABASE_DSN='user="PROD_USER" password="prod_password" connectString="prod:1521/PRODDB"'
go run cmd/migrate/main.go -env=production -direction=up
```

### 3. 애플리케이션 시작 시 자동 마이그레이션 (필수)

애플리케이션 시작 시 자동으로 마이그레이션을 실행하도록 설정합니다. 이를 통해 항상 최신 스키마로 실행됩니다.

**파일**: `cmd/api-bridge/main.go`

```go
import (
	"database/sql"

	_ "github.com/godror/godror"
	migrate "github.com/rubenv/sql-migrate"
)

func initializeDependencies(cfg *config.Config) (*Dependencies, error) {
	// 로거 초기화
	log := logger.NewLogger()

	// ... 기존 코드 ...

	// 🔥 데이터베이스 마이그레이션 실행 (필수)
	log.Info("Running database migrations...")
	if err := runMigrations(cfg, log); err != nil {
		// 마이그레이션 실패 시 애플리케이션 시작 중단
		return nil, fmt.Errorf("database migration failed: %w", err)
	}
	log.Info("✅ Database migrations completed successfully")

	// 데이터베이스 리포지토리 초기화 (마이그레이션 후)
	oracleRoutingRepo, err := database.NewOracleRoutingRepository(&cfg.Database)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to connect to OracleDB: %v", err))
		// ... Mock 리포지토리 사용
	}

	// ... 나머지 코드 ...
}

// runMigrations는 데이터베이스 마이그레이션을 실행합니다
func runMigrations(cfg *config.Config, log port.Logger) error {
	// 데이터베이스 연결
	dsn := cfg.Database.GetDSN()
	db, err := sql.Open("godror", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 연결 테스트
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// 마이그레이션 소스 설정
	migrations := &migrate.FileMigrationSource{
		Dir: "db/migrations",
	}

	// 마이그레이션 실행
	n, err := migrate.Exec(db, "oracle", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("failed to execute migrations: %w", err)
	}

	if n > 0 {
		log.Info(fmt.Sprintf("Applied %d new migration(s)", n))
	} else {
		log.Info("No new migrations to apply (schema is up-to-date)")
	}

	return nil
}
```

#### 환경별 마이그레이션 전략

**Development 환경**:
- ✅ 자동 마이그레이션 활성화
- 애플리케이션 시작 시 자동으로 최신 스키마 적용
- 마이그레이션 실패 시 애플리케이션 시작 중단

**Staging 환경**:
- ✅ 자동 마이그레이션 활성화
- 배포 전 스키마 자동 동기화
- 문제 발생 시 빠른 피드백

**Production 환경**:
- ⚠️ 환경 변수로 제어 가능하도록 설정
- 기본값: 자동 마이그레이션 활성화
- 대규모 마이그레이션 필요 시 수동 실행 옵션 제공

#### Config 설정 추가

**파일**: `pkg/config/config.go`

```go
type DatabaseConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	User         string        `yaml:"user"`
	Password     string        `yaml:"password"`
	ServiceName  string        `yaml:"service_name"`
	MaxOpenConns int           `yaml:"max_open_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns"`

	// 마이그레이션 설정 (기본값: true)
	AutoMigrate  bool          `yaml:"auto_migrate"`
}
```

**파일**: `config/config.yaml`

```yaml
database:
  host: localhost
  port: 1521
  user: DEMO_USER
  password: demo_password
  service_name: XEPDB1
  max_open_conns: 25
  max_idle_conns: 5

  # 자동 마이그레이션 설정
  # development/staging: true (권장)
  # production: 환경 변수로 제어 (기본값 true)
  auto_migrate: true
```

#### 프로덕션 환경에서 수동 제어

프로덕션 배포 시 대규모 마이그레이션이 필요한 경우:

```bash
# 1. 자동 마이그레이션 비활성화 (환경 변수)
export AUTO_MIGRATE=false

# 2. 애플리케이션 시작 전 수동 마이그레이션 실행
go run cmd/migrate/main.go -env=production -direction=up

# 3. 애플리케이션 시작
go run cmd/api-bridge/main.go
```

**장점**:
- ✅ 개발 환경에서 스키마 불일치 방지
- ✅ 팀원 간 스키마 동기화 자동화
- ✅ CI/CD 파이프라인에서 자동 스키마 업데이트
- ✅ 배포 프로세스 간소화

---

## 베스트 프랙티스

### 1. 마이그레이션 파일 작성 원칙

#### ✅ 해야 할 것

- **멱등성(Idempotent) 보장**: 여러 번 실행해도 안전하게 작성
  ```sql
  -- 좋은 예
  CREATE TABLE IF NOT EXISTS users (...);

  -- Oracle에서는 PL/SQL 블록 사용
  BEGIN
    EXECUTE IMMEDIATE 'CREATE TABLE users (...)';
  EXCEPTION
    WHEN OTHERS THEN
      IF SQLCODE != -955 THEN -- ORA-00955: name already used
        RAISE;
      END IF;
  END;
  ```

- **원자성(Atomic) 유지**: 하나의 마이그레이션은 하나의 논리적 변경만 수행
  ```sql
  -- 좋은 예: 하나의 테이블 생성
  -- +migrate Up
  CREATE TABLE users (...);

  -- 나쁜 예: 여러 테이블을 한 번에 생성
  -- +migrate Up
  CREATE TABLE users (...);
  CREATE TABLE orders (...);
  CREATE TABLE products (...);
  ```

- **항상 Down 마이그레이션 작성**: 롤백 가능하도록 작성
  ```sql
  -- +migrate Down
  DROP TABLE users CASCADE CONSTRAINTS;
  ```

- **테스트 데이터 분리**: 마이그레이션에 테스트 데이터 포함하지 않기
  - 별도의 seed 파일 사용 (`db/seeds/`)

#### ❌ 하지 말아야 할 것

- ❌ 이미 적용된 마이그레이션 수정 (새로운 마이그레이션 작성)
- ❌ 데이터 손실 위험이 있는 변경 (백업 필수)
- ❌ 프로덕션 데이터를 마이그레이션에 포함
- ❌ 외부 의존성 없이 실행 불가능한 마이그레이션

### 2. 버전 관리

```bash
# 마이그레이션 파일은 Git에 포함
git add db/migrations/*.sql

# dbconfig.yml은 .gitignore에 추가 (민감 정보 포함)
echo "dbconfig.yml" >> .gitignore

# 대신 템플릿 파일 제공
cp dbconfig.yml dbconfig.example.yml
git add dbconfig.example.yml
```

### 3. 팀 협업 시 충돌 방지

- **타임스탬프 기반 명명**: `YYYYMMDD_NNN_description.sql`
- **마이그레이션 순서 조율**: 팀원 간 번호 중복 방지
- **병합 전 확인**: Pull 받은 후 `sql-migrate status`로 확인

### 4. 대용량 데이터 마이그레이션

```sql
-- +migrate Up
-- 배치 처리로 대용량 데이터 마이그레이션
DECLARE
  CURSOR c_old_data IS SELECT * FROM old_table;
  TYPE t_data IS TABLE OF old_table%ROWTYPE INDEX BY PLS_INTEGER;
  v_data t_data;
  v_batch_size NUMBER := 1000;
BEGIN
  OPEN c_old_data;
  LOOP
    FETCH c_old_data BULK COLLECT INTO v_data LIMIT v_batch_size;
    EXIT WHEN v_data.COUNT = 0;

    FORALL i IN 1..v_data.COUNT
      INSERT INTO new_table (...)
      VALUES (v_data(i)...);

    COMMIT; -- 배치마다 커밋
  END LOOP;
  CLOSE c_old_data;
END;
/
```

### 5. 보안

```bash
# 프로덕션 환경은 환경 변수 사용
export DATABASE_DSN='user="prod_user" password="secure_password" connectString="prod:1521/PRODDB"'

# Kubernetes Secret 예시
kubectl create secret generic db-migration-secret \
  --from-literal=dsn='user="prod_user" password="secure_password" connectString="prod:1521/PRODDB"'
```

---

## 트러블슈팅

### 1. Oracle Instant Client 관련 오류

**증상**:
```
ORA-01804: failure to initialize timezone information
```

**해결 방법**:
```bash
# Windows
set ORA_TZFILE=C:\oracle\instantclient_21_3\timezone_35.dat

# Linux/Mac
export ORA_TZFILE=/opt/oracle/instantclient_21_3/timezone_35.dat
```

### 2. "name already used" 오류 (ORA-00955)

**증상**: 테이블이 이미 존재하여 마이그레이션 실패

**해결 방법**:
```sql
-- PL/SQL 블록으로 예외 처리
BEGIN
  EXECUTE IMMEDIATE 'CREATE TABLE users (...)';
EXCEPTION
  WHEN OTHERS THEN
    IF SQLCODE = -955 THEN
      DBMS_OUTPUT.PUT_LINE('Table already exists, skipping...');
    ELSE
      RAISE;
    END IF;
END;
/
```

### 3. schema_migrations 테이블 수동 초기화

```sql
-- 마이그레이션 이력 테이블 생성
CREATE TABLE schema_migrations (
    id VARCHAR2(255) NOT NULL PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 마이그레이션 이력 조회
SELECT * FROM schema_migrations ORDER BY applied_at;

-- 특정 마이그레이션 이력 삭제 (재실행 필요 시)
DELETE FROM schema_migrations WHERE id = '20250105_001_create_routing_rules.sql';
```

### 4. 마이그레이션 충돌 해결

**시나리오**: 여러 개발자가 동시에 마이그레이션 추가

```bash
# 1. 최신 코드 Pull
git pull origin main

# 2. 마이그레이션 상태 확인
sql-migrate status -config=dbconfig.yml -env=development

# 3. 누락된 마이그레이션 적용
sql-migrate up -config=dbconfig.yml -env=development

# 4. 충돌하는 마이그레이션 파일명 변경 (타임스탬프 조정)
mv 20250105_003_my_migration.sql 20250105_005_my_migration.sql
```

### 5. 롤백 후 데이터 복구

```sql
-- 롤백 전 백업 (필수!)
CREATE TABLE routing_rules_backup AS SELECT * FROM routing_rules;

-- 롤백 실행
-- sql-migrate down -limit=1

-- 복구 (필요 시)
INSERT INTO routing_rules SELECT * FROM routing_rules_backup;
```

---

## 참고 자료

### 공식 문서
- [sql-migrate GitHub](https://github.com/rubenv/sql-migrate)
- [godror 드라이버 문서](https://github.com/godror/godror)
- [Oracle SQL 가이드](https://docs.oracle.com/en/database/)

### 관련 가이드
- [GOLANG_SETUP_GUIDE.md](./GOLANG_SETUP_GUIDE.md) - Go 개발 환경 설정
- [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) - 배포 가이드
- [TESTING_GUIDE.md](./TESTING_GUIDE.md) - 테스트 작성 가이드

---

## 체크리스트

### 1. 초기 설정
- [x] sql-migrate 라이브러리 설치 (`go get -tags oracle github.com/rubenv/sql-migrate`)
- [ ] CLI 도구 설치 (선택사항) (`go install -tags oracle github.com/rubenv/sql-migrate/...@latest`)
- [x] `db/migrations/` 디렉토리 생성
- [x] `dbconfig.yml` 작성 (config.yaml의 database 설정 기반)
- [x] `.gitignore`에 `dbconfig.yml` 추가 확인
- [x] `dbconfig.example.yml` 템플릿 생성 완료 확인

### 2. Config 설정 추가
- [x] `pkg/config/config.go`에 `AutoMigrate bool` 필드 추가
- [x] `config/config.yaml`에 `auto_migrate: true` 설정 추가
- [x] 환경 변수 `AUTO_MIGRATE` 지원 구현

### 3. 마이그레이션 CLI 도구 구현
- [x] `cmd/migrate/main.go` 파일 생성
- [x] 환경별 DSN 설정 함수 구현 (`getDSNByEnv`)
- [x] 마이그레이션 실행 로직 구현 (up/down)
- [x] 환경 변수 우선 처리 로직 추가
- [x] CLI 테스트 (`go run cmd/migrate/main.go -env=development -direction=up`)

### 4. 애플리케이션 자동 마이그레이션 구현
- [x] `cmd/api-bridge/main.go`에 `runMigrations` 함수 추가
- [x] `initializeDependencies`에 자동 마이그레이션 호출 추가
- [x] 마이그레이션 실패 시 애플리케이션 시작 중단 로직 구현
- [x] 마이그레이션 성공/실패 로깅 추가
- [x] DB 연결 전 마이그레이션 실행 순서 확인

### 5. 마이그레이션 파일 작성
- [x] `20250105_001_create_routing_rules.sql` 작성
- [x] `20250105_002_create_api_endpoints.sql` 작성
- [x] `20250105_003_create_orchestration_rules.sql` 작성
- [x] `20250105_004_create_comparison_logs.sql` 작성
- [x] `20250105_005_add_performance_indexes.sql` 작성
- [x] 각 파일에 Up/Down 섹션 모두 작성
- [x] 멱등성 보장 (여러 번 실행 가능)
- [x] 외래 키 제약 조건 추가

### 6. go.mod 업데이트
- [x] `go get -tags oracle github.com/rubenv/sql-migrate` 실행
- [x] `go mod tidy` 실행하여 의존성 정리
- [x] `go.sum` 업데이트 확인

### 7. 로컬 환경 테스트
- [x] OracleDB 연결 확인 (dbconfig.yml 설정 검증)
- [x] CLI 도구로 마이그레이션 테스트 (`go run cmd/migrate/main.go -env=development -direction=up`)
- [x] 테이블 생성 확인 (5개 테이블: routing_rules, api_endpoints, orchestration_rules, comparison_logs, gorp_migrations)
- [x] 인덱스 생성 확인 (복합 인덱스 포함)
- [x] 외래 키 제약 조건 확인 (ON DELETE CASCADE 동작)
- [x] 롤백 테스트 (`go run cmd/migrate/main.go -env=development -direction=down -limit=1`)
- [x] 재실행 테스트 (멱등성 검증 - "No migrations to apply" 메시지 확인)

### 8. 애플리케이션 통합 테스트
- [x] 애플리케이션 시작 시 자동 마이그레이션 동작 확인
- [x] 마이그레이션 로그 출력 확인 ("Applied 5 new migration(s)" 메시지)
- [x] 기존 테이블이 있을 때 정상 동작 확인 ("No new migrations to apply" 메시지)
- [x] 마이그레이션 실패 시 애플리케이션 종료 확인 (fail-fast 동작)
- [x] API 엔드포인트 동작 테스트 (서버 정상 시작 확인)

### 9. 실행 전 확인 (배포)
- [ ] 백업 완료 (프로덕션 환경)
- [ ] 팀원에게 공지 (다운타임 필요 시)
- [ ] `sql-migrate status`로 현재 상태 확인
- [ ] 마이그레이션 파일 검토 (코드 리뷰)
- [ ] Staging 환경에서 선행 테스트 완료

### 10. 실행 후 검증
- [ ] 모든 테이블 생성 확인
- [ ] 인덱스 생성 확인
- [ ] 제약 조건 정상 동작 확인
- [ ] 애플리케이션 정상 구동 확인
- [ ] API 기능 테스트 (통합 테스트)
- [ ] 성능 테스트 (쿼리 실행 계획 확인)
- [ ] 로그 모니터링 (에러 없음 확인)

### 11. 문서화 및 정리
- [ ] 마이그레이션 이력 문서화
- [ ] 팀원 공유 및 가이드 전달
- [ ] 트러블슈팅 사례 업데이트
- [ ] Git 커밋 및 푸시

---

**작성일**: 2025-01-05
**작성자**: API Bridge Team
**버전**: 1.0.0
