# Swagger 문서 설정 가이드

## 개요

이 프로젝트는 `swagger.yaml` 파일 기반으로 API 문서를 제공합니다. 
코드 주석이 아닌 독립적인 YAML 파일을 통해 API 문서를 관리하여 유지보수성을 향상시켰습니다.

## 아키텍처

### 변경 전 (Annotation 기반)
```
handler.go (@Summary, @Description 주석)
    ↓
swag init (자동 생성)
    ↓
api-docs/docs.go (자동 생성 파일)
    ↓
Swagger UI
```

### 변경 후 (YAML 기반)
```
api-docs/swagger.yaml (수동 작성)
    ↓
Swagger UI (직접 참조)
```

## 파일 구조

```
demo-api-bridge/
├── api-docs/
│   └── swagger.yaml          # Swagger 문서 정의 (메인)
├── cmd/api-bridge/
│   └── main.go               # Swagger UI 설정
└── internal/adapter/inbound/http/
    └── handler.go            # 핸들러 (주석 제거됨)
```

## Swagger UI 접근

서비스 실행 후 다음 URL로 접근할 수 있습니다:

- **Swagger UI**: http://localhost:10019/swagger/index.html
- **Swagger YAML**: http://localhost:10019/swagger-yaml/swagger.yaml

## API 문서 업데이트 방법

### 1. swagger.yaml 파일 직접 수정

`api-docs/swagger.yaml` 파일을 직접 수정하여 API 문서를 업데이트합니다.

```yaml
# 새로운 엔드포인트 추가 예시
paths:
  /api/v1/new-endpoint:
    get:
      summary: 새로운 엔드포인트
      description: 새로운 기능 설명
      tags:
        - new-feature
      responses:
        "200":
          description: 성공
          schema:
            type: object
```

### 2. 변경사항 확인

1. 서비스 재시작 (hot-reload 미지원)
2. Swagger UI 페이지 새로고침
3. 변경된 API 문서 확인

## 장점

### 1. 코드와 문서의 분리
- 핸들러 코드가 깔끔해짐 (주석 제거)
- API 문서를 독립적으로 관리 가능
- 코드 변경 없이 문서만 수정 가능

### 2. 유지보수 용이
- YAML 파일 하나만 관리하면 됨
- 자동 생성 도구(swag) 의존성 제거
- 문서 변경 이력 관리 용이 (Git)

### 3. 팀 협업 개선
- 백엔드 개발자가 코드를 작성하는 동안
- 프론트엔드 개발자나 문서 담당자가 API 문서를 동시에 작성 가능
- Merge conflict 최소화

## 현재 문서화된 API

swagger.yaml 파일에는 다음 API가 모두 문서화되어 있습니다:

### Health & Status
- GET /health - 헬스체크
- GET /ready - Readiness 체크
- GET /api/v1/status - 서비스 상태

### Bridge API
- GET/POST/PUT/DELETE /api/v1/bridge/{path} - API 브리지 요청

### Endpoints CRUD
- POST /api/v1/endpoints - 엔드포인트 생성
- GET /api/v1/endpoints - 엔드포인트 목록
- GET /api/v1/endpoints/{id} - 엔드포인트 조회
- PUT /api/v1/endpoints/{id} - 엔드포인트 수정
- DELETE /api/v1/endpoints/{id} - 엔드포인트 삭제

### Routing Rules CRUD
- POST /api/v1/routing-rules - 라우팅 규칙 생성
- GET /api/v1/routing-rules - 라우팅 규칙 목록
- GET /api/v1/routing-rules/{id} - 라우팅 규칙 조회
- PUT /api/v1/routing-rules/{id} - 라우팅 규칙 수정
- DELETE /api/v1/routing-rules/{id} - 라우팅 규칙 삭제

### Orchestration Rules
- POST /api/v1/orchestration-rules - 오케스트레이션 규칙 생성
- GET /api/v1/orchestration-rules/{id} - 오케스트레이션 규칙 조회
- PUT /api/v1/orchestration-rules/{id} - 오케스트레이션 규칙 수정
- GET /api/v1/orchestration-rules/{id}/evaluate-transition - 전환 평가
- POST /api/v1/orchestration-rules/{id}/execute-transition - 전환 실행

### System
- POST /api/v1/shutdown - Graceful Shutdown
- GET /metrics - Prometheus 메트릭

## 주의사항

### 1. 자동 생성 불가
- 코드 주석에서 자동으로 문서가 생성되지 않습니다
- swagger.yaml 파일을 직접 수정해야 합니다

### 2. 일관성 유지
- 실제 API 구현과 swagger.yaml의 내용이 일치해야 합니다
- 새로운 API 추가 시 반드시 swagger.yaml도 업데이트하세요

### 3. 서비스 재시작 필요
- swagger.yaml 파일 수정 후 서비스를 재시작해야 반영됩니다
- 개발 환경에서는 air 등의 hot-reload 도구를 사용하는 것을 권장합니다

## 기존 swag 명령어

기존에 사용하던 swag 명령어는 더 이상 필요하지 않습니다:

```bash
# ❌ 더 이상 필요없음
# swag init -g cmd/api-bridge/main.go -o api-docs
```

## 예제: 새로운 API 추가하기

### 1. 핸들러 함수 작성 (handler.go)

```go
// GetUserProfile은 사용자 프로필을 조회합니다.
func (h *Handler) GetUserProfile(c *gin.Context) {
    userID := c.Param("id")
    // ... 구현
}
```

### 2. 라우트 등록 (main.go)

```go
router.GET("/api/v1/users/:id", handler.GetUserProfile)
```

### 3. Swagger 문서 작성 (swagger.yaml)

```yaml
/api/v1/users/{id}:
  get:
    summary: 사용자 프로필 조회
    description: 특정 사용자의 프로필 정보를 조회합니다
    tags:
      - users
    parameters:
      - name: id
        in: path
        description: 사용자 ID
        required: true
        type: string
    responses:
      "200":
        description: 프로필 조회 성공
        schema:
          type: object
          properties:
            id:
              type: string
            name:
              type: string
            email:
              type: string
      "404":
        description: 사용자를 찾을 수 없음
```

## 트러블슈팅

### Swagger UI가 로드되지 않을 때

1. 서비스가 정상적으로 실행되고 있는지 확인
2. `/swagger-yaml/swagger.yaml` URL이 정상적으로 응답하는지 확인
3. 브라우저 콘솔에서 오류 메시지 확인

### swagger.yaml 파일이 업데이트되지 않을 때

1. 서비스를 재시작했는지 확인
2. 브라우저 캐시를 지우고 새로고침 (Ctrl+Shift+R)
3. swagger.yaml 파일의 문법 오류 확인

### YAML 문법 검증

온라인 YAML 검증 도구를 사용하여 문법 오류를 확인할 수 있습니다:
- https://www.yamllint.com/
- https://editor.swagger.io/ (Swagger 전용)

## 참고 자료

- [Swagger 2.0 명세](https://swagger.io/specification/v2/)
- [OpenAPI 3.0 명세](https://swagger.io/specification/)
- [Swagger Editor](https://editor.swagger.io/)

