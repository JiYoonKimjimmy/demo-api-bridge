# 데이터베이스 마이그레이션 이력

API Bridge 프로젝트의 데이터베이스 스키마 마이그레이션 실행 이력입니다.

---

## 📅 마이그레이션 타임라인

### 2025-01-05: 초기 스키마 마이그레이션 (001~005)

#### Migration 001: Routing Rules 테이블 생성
**파일**: `db/migrations/20250105_001_create_routing_rules.sql`
**작성일**: 2025-01-05
**상태**: ✅ Development 적용, ✅ Staging 적용

**목적**:
- API 라우팅 규칙을 관리하는 핵심 테이블 생성
- 요청 경로, HTTP 메서드, 전략별 라우팅 정보 저장

**주요 컬럼**:
- `id` (VARCHAR2(36)): Primary Key
- `endpoint_id`: 연결된 API 엔드포인트 ID
- `request_path`: 요청 경로 (최대 500자)
- `method`: HTTP 메서드 (GET, POST, PUT, DELETE, PATCH)
- `strategy`: 라우팅 전략 (direct, orchestration, comparison, ab_test)
- `priority`: 우선순위 (NUMBER(10))
- `is_active`: 활성화 여부 (0/1)

**제약 조건**:
- CHECK: method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')
- CHECK: strategy IN ('direct', 'orchestration', 'comparison', 'ab_test')
- CHECK: is_active IN (0, 1)

**인덱스**:
- `idx_routing_path`: request_path
- `idx_routing_endpoint`: endpoint_id
- `idx_routing_active`: is_active

**영향**:
- 라우팅 규칙 동적 관리 가능
- 다중 전략 지원 (Direct, Orchestration, Comparison, A/B Test)

---

#### Migration 002: API Endpoints 테이블 생성
**파일**: `db/migrations/20250105_002_create_api_endpoints.sql`
**작성일**: 2025-01-05
**상태**: ✅ Development 적용, ✅ Staging 적용

**목적**:
- 외부 API 엔드포인트 정보를 저장
- 타임아웃, 재시도, 헤더 설정 관리

**주요 컬럼**:
- `id` (VARCHAR2(36)): Primary Key
- `name`: 엔드포인트 이름
- `base_url`: 기본 URL
- `path`: API 경로
- `method`: HTTP 메서드
- `timeout_ms`: 타임아웃 (기본값 5000ms)
- `retry_count`: 재시도 횟수 (기본값 3)
- `headers`: HTTP 헤더 (CLOB, JSON 형식)
- `is_active`: 활성화 여부

**제약 조건**:
- CHECK: method IN ('GET', 'POST', 'PUT', 'DELETE', 'PATCH')
- CHECK: is_active IN (0, 1)

**인덱스**:
- `idx_ep_name`: name
- `idx_ep_active`: is_active

**영향**:
- 외부 API 엔드포인트 설정 중앙 관리
- 타임아웃 및 재시도 전략 커스터마이징

---

#### Migration 003: Orchestration Rules 테이블 생성
**파일**: `db/migrations/20250105_003_create_orchestration_rules.sql`
**작성일**: 2025-01-05
**상태**: ✅ Development 적용, ✅ Staging 적용

**목적**:
- 다중 API 호출 오케스트레이션 규칙 관리
- 순차 실행(Sequential) 및 병렬 실행(Parallel) 지원

**주요 컬럼**:
- `id` (VARCHAR2(36)): Primary Key
- `routing_rule_id`: 연결된 라우팅 규칙 ID (FK)
- `name`: 오케스트레이션 규칙 이름
- `execution_type`: 실행 타입 (sequential, parallel)
- `steps`: 실행 스텝 (CLOB, JSON 배열)
- `is_active`: 활성화 여부

**제약 조건**:
- FK: `routing_rule_id` → `routing_rules(id)` ON DELETE CASCADE
- CHECK: execution_type IN ('sequential', 'parallel')
- CHECK: is_active IN (0, 1)

**인덱스**:
- `idx_orc_routing`: routing_rule_id
- `idx_orc_name`: name

**영향**:
- 복잡한 API 오케스트레이션 지원
- 레거시 시스템 통합 시 다중 호출 패턴 구현

---

#### Migration 004: Comparison Logs 테이블 생성
**파일**: `db/migrations/20250105_004_create_comparison_logs.sql`
**작성일**: 2025-01-05
**상태**: ✅ Development 적용, ✅ Staging 적용

**목적**:
- 레거시 API와 신규 API 응답 비교 로그 저장
- A/B 테스트 및 마이그레이션 검증 지원

**주요 컬럼**:
- `id` (VARCHAR2(36)): Primary Key
- `routing_rule_id`: 연결된 라우팅 규칙 ID (FK)
- `request_id`: 요청 추적 ID
- `old_response`: 레거시 API 응답 (CLOB)
- `new_response`: 신규 API 응답 (CLOB)
- `is_matched`: 응답 일치 여부 (0/1)
- `difference_details`: 차이점 상세 (CLOB, JSON 형식)
- `created_at`: 생성 시각

**제약 조건**:
- FK: `routing_rule_id` → `routing_rules(id)` ON DELETE CASCADE
- CHECK: is_matched IN (0, 1)

**인덱스**:
- `idx_cmp_routing`: routing_rule_id
- `idx_cmp_created`: created_at
- `idx_cmp_matched`: is_matched

**영향**:
- 레거시 시스템 마이그레이션 검증 데이터 수집
- API 응답 불일치 분석 및 모니터링

**참고**:
- 대용량 로그 관리를 위한 Partitioning 옵션 주석 처리됨 (필요 시 활성화)

---

#### Migration 005: 성능 최적화 인덱스 추가
**파일**: `db/migrations/20250105_005_add_performance_indexes.sql`
**작성일**: 2025-01-05
**상태**: ✅ Development 적용, ✅ Staging 적용
**특이사항**: ⚠️ DBMS_STATS 호출 주석 처리됨 (수동 실행 권장)

**목적**:
- 쿼리 성능 최적화를 위한 복합 인덱스 생성
- Oracle Optimizer 통계 정보 수집

**추가된 복합 인덱스**:
- `idx_routing_path_method`: routing_rules(request_path, method, is_active)
- `idx_ep_url_method`: api_endpoints(base_url, method, is_active)

**영향**:
- 다중 조건 쿼리 성능 향상 (예: 경로 + 메서드 조회)
- 활성 상태 필터링 쿼리 최적화

**트러블슈팅**:
- 문제: PL/SQL EXEC 문이 sql-migrate에서 파싱 오류 발생
- 해결: DBMS_STATS 호출 주석 처리
- 권장: Production 배포 시 수동으로 통계 수집 실행
  ```sql
  EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'ROUTING_RULES');
  EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'API_ENDPOINTS');
  EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'ORCHESTRATION_RULES');
  EXEC DBMS_STATS.GATHER_TABLE_STATS('DEMO_USER', 'COMPARISON_LOGS');
  ```

---

## 🌍 환경별 적용 이력

### Development 환경
**적용일**: 2025-01-05
**데이터베이스**: localhost:1521/XEPDB1
**사용자**: DEMO_USER

**실행 결과**:
```
✅ Applied 5 migration(s) successfully
```

**검증 결과**:
- ✅ 5개 테이블 생성 확인
  - ROUTING_RULES
  - API_ENDPOINTS
  - ORCHESTRATION_RULES
  - COMPARISON_LOGS
  - GORP_MIGRATIONS (마이그레이션 이력 테이블)

- ✅ 12개 인덱스 생성 확인
  - ROUTING_RULES: 4개 (idx_routing_path, idx_routing_endpoint, idx_routing_active, idx_routing_path_method)
  - API_ENDPOINTS: 3개 (idx_ep_name, idx_ep_active, idx_ep_url_method)
  - ORCHESTRATION_RULES: 2개 (idx_orc_routing, idx_orc_name)
  - COMPARISON_LOGS: 3개 (idx_cmp_routing, idx_cmp_created, idx_cmp_matched)

- ✅ 10개 제약 조건 생성 확인
  - Foreign Key: 2개 (FK_ORC_ROUTING, FK_CMP_ROUTING)
  - Check Constraint: 8개

---

### Staging 환경
**적용일**: 2025-01-12
**데이터베이스**: dev3-db.konadc.com:15321/kmdbp
**사용자**: DEMO_USER

**실행 명령어**:
```bash
go run cmd/migrate/main.go -env=staging -direction=up
```

**실행 결과**:
```
✅ Connected to database (staging)
✅ Applied 5 migration(s) (up)!
```

**검증 결과** (자동 검증 도구 사용):

1. **테이블 검증** (`go run cmd/verify/tables.go -env=staging`)
   ```
   ✅ ROUTING_RULES: 0 rows
   ✅ API_ENDPOINTS: 0 rows
   ✅ ORCHESTRATION_RULES: 0 rows
   ✅ COMPARISON_LOGS: 0 rows
   ✅ GORP_MIGRATIONS: 5 rows (migration history)
   ```

2. **인덱스 검증** (`go run cmd/verify/indexes.go -env=staging`)
   ```
   ✅ All 12 indexes are VALID
   - ROUTING_RULES: 4 indexes
   - API_ENDPOINTS: 3 indexes
   - ORCHESTRATION_RULES: 2 indexes
   - COMPARISON_LOGS: 3 indexes
   ```

3. **제약 조건 검증** (`go run cmd/verify/constraints.go -env=staging`)
   ```
   ✅ All 10 constraints are ENABLED
   - Foreign Keys: 2 (ON DELETE CASCADE verified)
   - Check Constraints: 8
   ```

**롤백 테스트**:
```bash
# 1개 마이그레이션 롤백
go run cmd/migrate/main.go -env=staging -direction=down -limit=1
✅ Applied 1 migration(s) (down)!

# 재적용
go run cmd/migrate/main.go -env=staging -direction=up
✅ Applied 1 migration(s) (up)!
```

**멱등성 검증**:
```bash
# 이미 적용된 상태에서 재실행
go run cmd/migrate/main.go -env=staging -direction=up
✅ No migrations to apply (schema is up-to-date)
```

---

### Production 환경
**상태**: ⏳ 배포 대기 중

**배포 계획**:
1. 프로덕션 데이터베이스 백업 (필수)
2. 점검 시간대 선정 및 팀원 공지
3. 마이그레이션 실행
4. 검증 도구로 스키마 확인
5. 애플리케이션 구동 테스트
6. 수동 DBMS_STATS 실행 (성능 최적화)

**배포 명령어** (예정):
```bash
# 환경 변수 설정
export DATABASE_DSN='user="DEMO_USER" password="<PROD_PASSWORD>" connectString="<PROD_HOST>:1521/<PROD_DB>"'

# 마이그레이션 실행
go run cmd/migrate/main.go -env=production -direction=up

# 검증
go run cmd/verify/tables.go -env=production
go run cmd/verify/indexes.go -env=production
go run cmd/verify/constraints.go -env=production
```

---

## 📊 마이그레이션 통계

### 전체 마이그레이션 현황

| Migration | 파일명 | Development | Staging | Production |
|-----------|--------|-------------|---------|------------|
| 001 | create_routing_rules | ✅ 2025-01-05 | ✅ 2025-01-12 | ⏳ 대기 |
| 002 | create_api_endpoints | ✅ 2025-01-05 | ✅ 2025-01-12 | ⏳ 대기 |
| 003 | create_orchestration_rules | ✅ 2025-01-05 | ✅ 2025-01-12 | ⏳ 대기 |
| 004 | create_comparison_logs | ✅ 2025-01-05 | ✅ 2025-01-12 | ⏳ 대기 |
| 005 | add_performance_indexes | ✅ 2025-01-05 | ✅ 2025-01-12 | ⏳ 대기 |

### 스키마 통계

**생성된 객체**:
- 테이블: 4개 (+ 1개 마이그레이션 이력 테이블)
- 인덱스: 12개 (단일 인덱스 10개 + 복합 인덱스 2개)
- 제약 조건: 10개 (Foreign Key 2개 + Check Constraint 8개)

**예상 스토리지**:
- 초기 빈 테이블: ~10 MB
- 1만 건 라우팅 규칙 기준: ~50 MB
- 1백만 건 비교 로그 기준: ~2 GB (Partitioning 권장)

---

## 🛠️ 도구 및 자동화

### 구현된 마이그레이션 도구

#### 1. 마이그레이션 CLI (`cmd/migrate/main.go`)
**기능**:
- 마이그레이션 적용 (up)
- 마이그레이션 롤백 (down)
- 마이그레이션 상태 확인 (status)
- 환경별 DSN 설정 (development, staging, production)
- 환경 변수 우선 지원 (DATABASE_DSN)

**사용 예시**:
```bash
# Up
go run cmd/migrate/main.go -env=development -direction=up

# Down (1개 롤백)
go run cmd/migrate/main.go -env=development -direction=down -limit=1

# Status
go run cmd/migrate/main.go -env=development -direction=status
```

#### 2. 테이블 검증 도구 (`cmd/verify/tables.go`)
**기능**:
- 모든 테이블 존재 여부 확인
- 테이블별 레코드 수 조회
- 마이그레이션 이력 확인

#### 3. 인덱스 검증 도구 (`cmd/verify/indexes.go`)
**기능**:
- 모든 인덱스 생성 확인
- 인덱스 상태 확인 (VALID/INVALID)
- 테이블별 인덱스 목록 조회

#### 4. 제약 조건 검증 도구 (`cmd/verify/constraints.go`)
**기능**:
- Foreign Key 제약 조건 확인
- Check 제약 조건 확인
- 제약 조건 상태 확인 (ENABLED/DISABLED)

### 자동 마이그레이션

**애플리케이션 시작 시 자동 실행**:
- 파일: `cmd/api-bridge/main.go`
- 기능: 애플리케이션 시작 시 자동으로 최신 스키마 적용
- 설정: `config.yaml`의 `auto_migrate: true` (기본값)

**장점**:
- ✅ 개발 환경에서 스키마 불일치 방지
- ✅ 팀원 간 스키마 자동 동기화
- ✅ CI/CD 파이프라인 간소화
- ✅ 배포 프로세스 자동화

---

## 📝 주요 이슈 및 해결

### Issue #1: DBMS_STATS 실행 문제
**문제**: Migration 005의 PL/SQL EXEC 문이 sql-migrate에서 파싱 오류 발생
**원인**: sql-migrate가 PL/SQL 블록의 EXEC 문을 SQL 문으로 인식하지 못함
**해결**: DBMS_STATS 호출을 주석 처리하고 수동 실행 권장
**커밋**: `e615de4 - fix: Simplify migration 005 by commenting out PL/SQL DBMS_STATS`

### Issue #2: Oracle 드라이버 변경
**변경**: godror → sijms/go-ora
**이유**: Windows 환경에서 Oracle Instant Client 설정 간소화
**영향**: DSN 형식 변경, 모든 마이그레이션 도구 업데이트
**상태**: 완료

### Issue #3: 마이그레이션 테이블 이름
**테이블명**: `gorp_migrations` (예상: `schema_migrations`)
**원인**: sql-migrate가 oci8 dialect에서 gorp_migrations 사용
**영향**: 없음 (정상 동작)
**확인**: Staging 환경에서 검증 완료

---

## 📚 관련 문서

- [DB_MIGRATION_GUIDE.md](./DB_MIGRATION_GUIDE.md) - 마이그레이션 전체 가이드
- [TEAM_ONBOARDING.md](./TEAM_ONBOARDING.md) - 팀원 온보딩 가이드
- [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) - 배포 가이드
- [README.md](../README.md) - 프로젝트 개요

---

**작성일**: 2025-01-12
**최종 수정**: 2025-01-12
**작성자**: API Bridge Team
**버전**: 1.0.0
