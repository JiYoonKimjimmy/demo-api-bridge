# 팀원 온보딩 가이드 - DB 마이그레이션

API Bridge 프로젝트의 신규 팀원을 위한 데이터베이스 마이그레이션 Quick Start 가이드입니다.

---

## 🚀 Quick Start (5분 완료)

### 1단계: 프로젝트 클론
```bash
git clone <repository-url>
cd demo-api-bridge
```

### 2단계: 데이터베이스 설정 파일 생성
```bash
# 템플릿 복사
cp dbconfig.example.yml dbconfig.yml

# 로컬 환경에 맞게 수정 (예: 비밀번호 변경)
# dbconfig.yml의 development 섹션 확인
```

### 3단계: 마이그레이션 실행
```bash
# 모든 마이그레이션 적용
go run cmd/migrate/main.go -env=development -direction=up
```

**예상 출력**:
```
✅ Connected to database (development)
✅ Applied 5 migration(s) (up)!
```

### 4단계: 검증
```bash
# 테이블 생성 확인
go run cmd/verify/tables.go -env=development
```

**예상 출력**:
```
✅ ROUTING_RULES: 0 rows
✅ API_ENDPOINTS: 0 rows
✅ ORCHESTRATION_RULES: 0 rows
✅ COMPARISON_LOGS: 0 rows
✅ GORP_MIGRATIONS: 5 rows
```

### 5단계: 애플리케이션 실행
```bash
# 애플리케이션 시작 (자동 마이그레이션 포함)
go run cmd/api-bridge/main.go
```

---

## 📋 사전 요구사항

### 필수 소프트웨어

#### 1. Go 1.23 이상
```bash
# 버전 확인
go version

# 예상 출력: go version go1.23.4 windows/amd64
```

**설치 필요 시**: [GOLANG_SETUP_GUIDE.md](./GOLANG_SETUP_GUIDE.md) 참조

#### 2. OracleDB 접근 권한
- **Development**: localhost:1521/XEPDB1
- **Staging**: dev3-db.konadc.com:15321/kmdbp
- **사용자**: DEMO_USER
- **비밀번호**: 팀 리더에게 문의

#### 3. Git
```bash
# 버전 확인
git --version
```

### 선택 사항

#### Oracle Instant Client (godror 사용 시)
- **현재 프로젝트**: sijms/go-ora 사용 (Instant Client 불필요)
- **만약 godror로 변경 시**: Oracle Instant Client 설치 필요

---

## 🛠️ 주요 명령어 치트시트

### 마이그레이션 CLI (`cmd/migrate/main.go`)

#### 기본 명령어
```bash
# 모든 마이그레이션 적용
go run cmd/migrate/main.go -env=development -direction=up

# 마이그레이션 상태 확인
go run cmd/migrate/main.go -env=development -direction=status

# 마지막 1개 롤백
go run cmd/migrate/main.go -env=development -direction=down -limit=1
```

#### 환경별 실행
```bash
# Development (로컬)
go run cmd/migrate/main.go -env=development -direction=up

# Staging (테스트 서버)
go run cmd/migrate/main.go -env=staging -direction=up

# Production (운영 서버) - 신중하게!
export DATABASE_DSN='oracle://DEMO_USER:<password>@<host>:1521/<db>'
go run cmd/migrate/main.go -env=production -direction=up
```

### 검증 도구

#### 1. 테이블 검증
```bash
go run cmd/verify/tables.go -env=development
```
**출력**: 5개 테이블 존재 확인 및 레코드 수

#### 2. 인덱스 검증
```bash
go run cmd/verify/indexes.go -env=development
```
**출력**: 12개 인덱스 VALID 상태 확인

#### 3. 제약 조건 검증
```bash
go run cmd/verify/constraints.go -env=development
```
**출력**: 10개 제약 조건 (FK 2개, CHECK 8개) ENABLED 상태 확인

### 애플리케이션 실행
```bash
# 개발 모드 (자동 마이그레이션 포함)
go run cmd/api-bridge/main.go

# 빌드 후 실행
go build -o api-bridge cmd/api-bridge/main.go
./api-bridge
```

### SQL*Plus 직접 접속 (디버깅용)
```bash
# Development 환경 접속
sqlplus DEMO_USER/<password>@localhost:1521/XEPDB1

# 마이그레이션 이력 확인
SELECT id, applied_at FROM gorp_migrations ORDER BY applied_at;

# 테이블 목록 확인
SELECT table_name FROM user_tables;

# 종료
EXIT;
```

---

## 🔍 자주 하는 작업

### 1. 새로운 마이그레이션 추가

```bash
# 1. 파일 생성 (명명 규칙 준수)
# 형식: YYYYMMDD_NNN_description.sql
touch db/migrations/20250115_006_add_user_column.sql

# 2. 마이그레이션 작성
cat > db/migrations/20250115_006_add_user_column.sql <<'EOF'
-- +migrate Up
ALTER TABLE routing_rules ADD user_id VARCHAR2(36);
CREATE INDEX idx_routing_user ON routing_rules(user_id);

-- +migrate Down
DROP INDEX idx_routing_user;
ALTER TABLE routing_rules DROP COLUMN user_id;
EOF

# 3. 적용
go run cmd/migrate/main.go -env=development -direction=up

# 4. 검증
go run cmd/verify/tables.go -env=development
```

### 2. 마이그레이션 롤백 및 재실행

```bash
# 1. 마지막 마이그레이션 롤백
go run cmd/migrate/main.go -env=development -direction=down -limit=1

# 2. 마이그레이션 파일 수정
# (db/migrations/20250115_006_add_user_column.sql 편집)

# 3. 재적용
go run cmd/migrate/main.go -env=development -direction=up
```

### 3. 다른 브랜치로 전환 후 스키마 동기화

```bash
# 1. 브랜치 전환
git checkout feature/new-api

# 2. 최신 코드 Pull
git pull origin feature/new-api

# 3. 마이그레이션 상태 확인
go run cmd/migrate/main.go -env=development -direction=status

# 4. 누락된 마이그레이션 자동 적용
go run cmd/migrate/main.go -env=development -direction=up

# 또는 애플리케이션 실행 (자동 마이그레이션)
go run cmd/api-bridge/main.go
```

### 4. 스키마 완전 초기화 (주의!)

```bash
# ⚠️ 경고: 모든 데이터 삭제됨!

# 1. 모든 마이그레이션 롤백
go run cmd/migrate/main.go -env=development -direction=down

# 2. 재적용
go run cmd/migrate/main.go -env=development -direction=up

# 또는 SQL*Plus로 수동 삭제
sqlplus DEMO_USER/<password>@localhost:1521/XEPDB1 <<EOF
DROP TABLE comparison_logs CASCADE CONSTRAINTS;
DROP TABLE orchestration_rules CASCADE CONSTRAINTS;
DROP TABLE routing_rules CASCADE CONSTRAINTS;
DROP TABLE api_endpoints CASCADE CONSTRAINTS;
DROP TABLE gorp_migrations CASCADE CONSTRAINTS;
EXIT;
EOF

# 재적용
go run cmd/migrate/main.go -env=development -direction=up
```

### 5. Production 배포 전 Staging 테스트

```bash
# 1. Staging 환경 마이그레이션
go run cmd/migrate/main.go -env=staging -direction=up

# 2. 전체 검증
go run cmd/verify/tables.go -env=staging
go run cmd/verify/indexes.go -env=staging
go run cmd/verify/constraints.go -env=staging

# 3. 애플리케이션 테스트
# (Staging 서버에서 애플리케이션 구동 및 API 테스트)

# 4. 문제 발생 시 롤백
go run cmd/migrate/main.go -env=staging -direction=down -limit=1
```

---

## ⚠️ 주의사항

### 절대 하지 말아야 할 것

1. ❌ **이미 적용된 마이그레이션 파일 수정 금지**
   - 이유: 다른 환경(Staging/Production)과 불일치 발생
   - 해결: 새로운 마이그레이션 파일 생성

2. ❌ **Production 환경에서 무작정 롤백 금지**
   - 이유: 데이터 손실 위험
   - 해결: 반드시 백업 후 롤백, 팀 리더 승인 필수

3. ❌ **마이그레이션 파일에 테스트 데이터 포함 금지**
   - 이유: Production 환경 오염
   - 해결: 별도 seed 파일 사용 (`db/seeds/`)

4. ❌ **비밀번호를 dbconfig.yml에 커밋 금지**
   - 이유: 보안 위험
   - 해결: .gitignore에 dbconfig.yml 추가됨 (확인 완료)

5. ❌ **외래 키 제약 조건 없이 테이블 삭제 금지**
   - 이유: 참조 무결성 위반
   - 해결: `CASCADE CONSTRAINTS` 옵션 사용

### 권장 사항

1. ✅ **항상 마이그레이션 상태 확인 후 작업**
   ```bash
   go run cmd/migrate/main.go -env=development -direction=status
   ```

2. ✅ **애플리케이션 시작 전 마이그레이션 자동 실행 활용**
   - `config.yaml`의 `auto_migrate: true` 설정 확인

3. ✅ **마이그레이션 작성 시 Down 섹션 필수 작성**
   - 롤백 가능하도록 항상 Down 마이그레이션 포함

4. ✅ **팀원과 마이그레이션 번호 조율**
   - 동일한 타임스탬프/번호 중복 방지

5. ✅ **검증 도구로 항상 결과 확인**
   ```bash
   go run cmd/verify/tables.go -env=development
   ```

---

## 🐛 트러블슈팅

### 문제 1: 마이그레이션 실행 시 연결 오류

**증상**:
```
❌ Failed to connect to database: ORA-12541: TNS:no listener
```

**해결 방법**:
1. OracleDB 서버 실행 상태 확인
   ```bash
   # Windows: Oracle 서비스 확인
   sc query OracleServiceXE

   # Linux/Mac: lsnrctl 확인
   lsnrctl status
   ```

2. `dbconfig.yml`의 연결 정보 확인
   ```yaml
   development:
     datasource: "oracle://DEMO_USER:demo_password@localhost:1521/XEPDB1"
   ```

3. 네트워크 방화벽 확인 (포트 1521 오픈)

### 문제 2: "No migrations to apply" 메시지

**증상**:
```
✅ Connected to database (development)
No migrations to apply (schema is up-to-date)
```

**의미**: 정상 상태 (모든 마이그레이션이 이미 적용됨)

**확인 방법**:
```bash
# 마이그레이션 상태 확인
go run cmd/migrate/main.go -env=development -direction=status

# 예상 출력:
# 20250105_001_create_routing_rules.sql: APPLIED
# 20250105_002_create_api_endpoints.sql: APPLIED
# ...
```

### 문제 3: "Table already exists" 오류

**증상**:
```
ORA-00955: name is already used by an existing object
```

**원인**: 테이블이 이미 존재하는데 마이그레이션이 기록되지 않음

**해결 방법**:
```sql
-- SQL*Plus로 수동 마이그레이션 이력 추가
sqlplus DEMO_USER/<password>@localhost:1521/XEPDB1

INSERT INTO gorp_migrations (id, applied_at)
VALUES ('20250105_001_create_routing_rules.sql', CURRENT_TIMESTAMP);

COMMIT;
EXIT;
```

### 문제 4: Go 모듈 의존성 오류

**증상**:
```
go: cannot find module providing package github.com/rubenv/sql-migrate
```

**해결 방법**:
```bash
# 의존성 다운로드
go mod download

# 또는 모듈 초기화
go mod tidy
```

### 문제 5: 권한 부족 오류

**증상**:
```
ORA-01031: insufficient privileges
```

**해결 방법**:
1. DBA에게 권한 요청
   - CREATE TABLE
   - CREATE INDEX
   - CREATE SEQUENCE (필요 시)

2. 권한 확인 (SQL*Plus)
   ```sql
   SELECT * FROM user_sys_privs WHERE privilege LIKE '%CREATE%';
   ```

---

## 📚 추가 학습 자료

### 필독 문서
1. [DB_MIGRATION_GUIDE.md](./DB_MIGRATION_GUIDE.md) - 마이그레이션 전체 가이드
2. [MIGRATION_HISTORY.md](./MIGRATION_HISTORY.md) - 마이그레이션 이력
3. [README.md](../README.md) - 프로젝트 개요

### 참고 자료
- [sql-migrate GitHub](https://github.com/rubenv/sql-migrate) - 공식 문서
- [sijms/go-ora GitHub](https://github.com/sijms/go-ora) - Oracle 드라이버
- [Oracle SQL 가이드](https://docs.oracle.com/en/database/) - Oracle 공식 문서

### 팀 커뮤니케이션
- **질문**: 팀 리더 또는 시니어 개발자에게 문의
- **이슈 공유**: GitHub Issues 또는 팀 채팅
- **배포 협의**: 반드시 팀 리더 승인 후 진행

---

## ✅ 온보딩 체크리스트

완료 후 체크하세요!

### 환경 설정
- [ ] Go 1.23 이상 설치 확인 (`go version`)
- [ ] 프로젝트 클론 완료
- [ ] `dbconfig.yml` 생성 및 설정
- [ ] OracleDB 연결 테스트 완료

### 마이그레이션 실행
- [ ] Development 환경 마이그레이션 적용
- [ ] 테이블 생성 확인 (5개)
- [ ] 인덱스 생성 확인 (12개)
- [ ] 제약 조건 확인 (10개)

### 애플리케이션 테스트
- [ ] 애플리케이션 정상 실행
- [ ] 자동 마이그레이션 동작 확인
- [ ] API 엔드포인트 테스트 (예: `/health`)

### 문서 숙지
- [ ] DB_MIGRATION_GUIDE.md 읽기
- [ ] TEAM_ONBOARDING.md (이 문서) 읽기
- [ ] 주요 명령어 치트시트 숙지

### 팀 협업
- [ ] 팀 리더에게 온보딩 완료 보고
- [ ] 팀 채팅방 가입
- [ ] 첫 번째 커밋 및 PR 생성

---

## 🆘 도움 요청

막히는 부분이 있으면 주저하지 말고 요청하세요!

1. **팀 리더**: [이름] - [이메일/슬랙]
2. **시니어 개발자**: [이름] - [이메일/슬랙]
3. **GitHub Issues**: [프로젝트 Issues 링크]
4. **팀 채팅**: [슬랙/MS Teams 채널]

**환영합니다! 🎉**

---

**작성일**: 2025-01-12
**최종 수정**: 2025-01-12
**작성자**: API Bridge Team
**버전**: 1.0.0
