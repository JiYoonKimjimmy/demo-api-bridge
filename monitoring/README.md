# API Bridge 모니터링 시스템

API Bridge 시스템의 모니터링을 위한 Prometheus, Grafana, AlertManager 설정입니다.

## 구성 요소

### 1. Prometheus
- **역할**: 메트릭 수집 및 저장
- **포트**: 9090
- **설정 파일**: `prometheus/prometheus.yml`
- **알림 규칙**: `prometheus/alerts.yml`

### 2. Grafana
- **역할**: 대시보드 및 시각화
- **포트**: 3000
- **기본 로그인**: admin / admin123
- **대시보드**: API Bridge 전용 대시보드 포함

### 3. AlertManager
- **역할**: 알림 관리 및 라우팅
- **포트**: 9093
- **설정 파일**: `alertmanager/alertmanager.yml`

### 4. Node Exporter
- **역할**: 시스템 메트릭 수집
- **포트**: 9100

## 시작하기

### 1. Docker Compose로 전체 스택 실행
```bash
cd monitoring
docker-compose up -d
```

### 2. 서비스 접속
- **Grafana**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **AlertManager**: http://localhost:9093

### 3. API Bridge 서비스 실행
```bash
# API Bridge 서비스를 실행하여 메트릭 생성
go run cmd/api-bridge/main.go
```

## 대시보드 구성

### API Bridge Dashboard
주요 메트릭들:
- **API Request Rate**: 초당 요청 수
- **Request Latency**: 응답 시간 (95th, 50th percentile)
- **Success Rate**: 성공률
- **API Comparison Match Rate**: API 비교 일치율
- **API Mode Transitions**: API 모드 전환 횟수
- **External API Call Rate**: 외부 API 호출률
- **Cache Hit Rate**: 캐시 히트율
- **Circuit Breaker Status**: Circuit Breaker 상태

## 알림 규칙

### Critical 알림
- **LowMatchRate**: API 비교 일치율이 90% 미만
- **CircuitBreakerOpen**: Circuit Breaker가 열린 상태

### Warning 알림
- **HighErrorRate**: 에러율이 5% 초과
- **HighLatency**: 95th percentile 지연시간이 500ms 초과
- **HighExternalAPIFailureRate**: 외부 API 실패율이 10% 초과
- **LowCacheHitRate**: 캐시 히트율이 50% 미만
- **FrequentModeTransitions**: API 모드 전환이 너무 빈번

## 커스터마이징

### 메트릭 추가
1. `prometheus/prometheus.yml`에서 새로운 타겟 추가
2. Grafana 대시보드에 새 패널 추가

### 알림 규칙 수정
1. `prometheus/alerts.yml`에서 규칙 수정
2. `alertmanager/alertmanager.yml`에서 라우팅 수정

### 대시보드 수정
1. Grafana 웹 UI에서 직접 수정
2. JSON 파일을 수정하여 재배포

## 트러블슈팅

### 메트릭이 표시되지 않는 경우
1. API Bridge 서비스가 실행 중인지 확인
2. Prometheus 타겟이 UP 상태인지 확인
3. 네트워크 연결 확인

### 알림이 작동하지 않는 경우
1. AlertManager 설정 확인
2. 이메일 서버 설정 확인
3. 알림 규칙 문법 확인

## 성능 최적화

### Prometheus 설정
- `scrape_interval` 조정으로 수집 빈도 조절
- `retention.time` 설정으로 데이터 보관 기간 조절

### Grafana 설정
- 대시보드 새로고침 간격 조정
- 패널 수 줄여서 성능 향상

## 보안

### 프로덕션 환경에서의 주의사항
1. 기본 패스워드 변경
2. 네트워크 접근 제한
3. SSL/TLS 인증서 설정
4. 방화벽 규칙 설정
