# API Bridge 시스템 - 배포 가이드

API Bridge 시스템의 배포 및 운영을 위한 프로세스 관리, 가상IP 설정, 배포 스크립트 가이드입니다.

---

## 방법 비교표

| 항목 | Systemd | Shell Script | Supervisor | PM2 | Screen/Tmux |
|------|---------|--------------|------------|-----|-------------|
| **설정 난이도** | 중간 | 쉬움 | 중간 | 쉬움 | 매우 쉬움 |
| **Root 권한** | 필요 | 불필요 | 필요 | 불필요 | 불필요 |
| **자동 재시작** | ✅ | ❌ | ✅ | ✅ | ❌ |
| **리소스 제한** | ✅ | ❌ (ulimit) | ✅ | ❌ | ❌ |
| **로그 관리** | journalctl | 파일 | 파일+로테이션 | 파일+로테이션 | 콘솔 |
| **부팅 시 자동 시작** | ✅ | cron 설정 | ✅ | ✅ | ❌ |
| **프로세스 모니터링** | ✅ | 수동 | ✅ + Web UI | ✅ + Web UI | 수동 |
| **로그 로테이션** | journald | logrotate | 내장 | 내장 | ❌ |
| **클러스터 모드** | ❌ | ❌ | ❌ | ✅ | ❌ |
| **학습 곡선** | 중간 | 낮음 | 중간 | 낮음 | 낮음 |
| **외부 의존성** | Systemd | 없음 | Python | Node.js | 없음 |
| **운영 환경 적합성** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐ |
| **배포 편의성** | 중간 | 높음 | 중간 | 높음 | 높음 |
| **장애 복구** | 자동 | 수동 | 자동 | 자동 | 수동 |
| **멀티 인스턴스** | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## 추천 시나리오

### 🎯 선택 가이드

#### 운영 환경

| 상황 | 추천 방법 | 이유 |
|------|----------|------|
| Root 권한 O + Linux 표준 | **Systemd** | 시스템 통합, 자동 재시작, 리소스 제한 |
| Root 권한 X | **Shell Script** | 간단한 배포, 일반 사용자 권한 |
| 웹 UI 필요 | **Supervisor** | 웹 대시보드로 모니터링 |
| Node.js 환경 | **PM2** | 클러스터 모드, 로드밸런싱 |

#### 개발/테스트 환경

| 상황 | 추천 방법 | 이유 |
|------|----------|------|
| 빠른 테스트 | **Screen/Tmux** | 실시간 로그, 세션 관리 |
| 로컬 개발 | **Shell Script** | 간단한 시작/중지 |
| CI/CD 테스트 | **Shell Script** | 자동화 스크립트 통합 |

---

## 상세 비교

### 1. Systemd

#### 장점
- ✅ Linux 표준 서비스 관리자
- ✅ 자동 재시작 (on-failure)
- ✅ 리소스 제한 (CPU, Memory)
- ✅ journalctl을 통한 중앙 로그 관리
- ✅ 부팅 시 자동 시작
- ✅ 의존성 관리 (After, Requires)

#### 단점
- ❌ Root 권한 필수
- ❌ Systemd가 없는 환경에서 사용 불가
- ❌ 설정 파일 작성 필요

#### 적합한 경우
- 프로덕션 환경
- 시스템 서비스로 관리 필요
- 자동 재시작 및 리소스 제한 필요

#### 예제
```ini
[Unit]
Description=API Bridge Service
After=network.target

[Service]
Type=simple
User=apibridge
ExecStart=/opt/api-bridge/bin/api-bridge --bind-address=192.168.1.101 --bind-port=10019
Restart=on-failure
MemoryLimit=512M

[Install]
WantedBy=multi-user.target
```

---

### 2. Shell Script ⭐ (프로젝트 선택)

#### 장점
- ✅ Root 권한 불필요
- ✅ 간단한 구조
- ✅ 빠른 배포
- ✅ 외부 의존성 없음
- ✅ 배포 스크립트와 쉬운 통합
- ✅ PID 기반 명확한 프로세스 관리

#### 단점
- ❌ 자동 재시작 없음 (Cron으로 보완 가능)
- ❌ 리소스 제한 기능 제한적 (ulimit만 가능)
- ❌ 수동 모니터링 필요

#### 적합한 경우
- Root 권한이 없는 환경
- 빠른 배포가 필요한 경우
- 간단한 프로세스 관리
- 가상IP 기반 다중 인스턴스

#### 프로세스 감시 (Cron)
자동 재시작이 필요한 경우 Cron으로 감시 스크립트 실행:

```bash
# watchdog.sh
#!/bin/bash
for instance in 1 2 3; do
    if ! curl -f "http://192.168.1.10${instance}:10019/health" > /dev/null 2>&1; then
        /opt/api-bridge/scripts/restart.sh $instance
    fi
done
```

```bash
# Cron 등록 (5분마다)
*/5 * * * * /opt/api-bridge/scripts/watchdog.sh
```

---

### 3. Supervisor

#### 장점
- ✅ 프로세스 관리 특화
- ✅ 웹 UI 제공 (http://localhost:9001)
- ✅ 자동 재시작
- ✅ 로그 로테이션 내장
- ✅ 여러 프로세스 그룹 관리
- ✅ RPC 인터페이스

#### 단점
- ❌ Python 의존성
- ❌ Root 권한 필요 (일반적으로)
- ❌ 추가 설치 필요

#### 적합한 경우
- 웹 UI로 모니터링 필요
- 여러 프로세스 통합 관리
- 자동 재시작 필수

#### 예제
```ini
[program:api-bridge-1]
command=/opt/api-bridge/bin/api-bridge --bind-address=192.168.1.101 --bind-port=10019
autostart=true
autorestart=true
user=apibridge
stdout_logfile=/var/log/api-bridge-1.log

[group:api-bridge]
programs=api-bridge-1,api-bridge-2,api-bridge-3
```

---

### 4. PM2

#### 장점
- ✅ 클러스터 모드 지원
- ✅ 로드밸런싱 내장
- ✅ 자동 재시작
- ✅ 웹 대시보드 (PM2 Plus)
- ✅ 로그 관리 및 로테이션
- ✅ 배포 기능 (ecosystem.config.js)

#### 단점
- ❌ Node.js 의존성
- ❌ Go 바이너리 실행 시 일부 기능 제한
- ❌ 추가 설치 필요

#### 적합한 경우
- Node.js 환경
- 클러스터 모드 필요
- 웹 대시보드 선호

#### 예제
```bash
# 시작
pm2 start /opt/api-bridge/bin/api-bridge --name api-bridge-1 -- --bind-address=192.168.1.101 --bind-port=10019

# 클러스터 모드
pm2 start app.js -i max

# 모니터링
pm2 monit

# 상태 확인
pm2 status
```

---

### 5. Screen/Tmux

#### 장점
- ✅ 매우 간단
- ✅ 실시간 로그 확인
- ✅ 세션 유지
- ✅ 외부 의존성 최소
- ✅ SSH 연결 끊어져도 프로세스 유지

#### 단점
- ❌ 자동 재시작 없음
- ❌ 운영 환경 부적합
- ❌ 모니터링 기능 없음
- ❌ 리소스 제한 없음

#### 적합한 경우
- 개발/테스트 환경
- 임시 실행
- 실시간 디버깅

#### 예제

**Screen**:
```bash
# 시작
screen -dmS api-bridge-1 /opt/api-bridge/bin/api-bridge --bind-address=192.168.1.101 --bind-port=10019

# 접속
screen -r api-bridge-1

# Detach: Ctrl+A, D
```

**Tmux**:
```bash
# 시작
tmux new-session -d -s api-bridge-1 '/opt/api-bridge/bin/api-bridge --bind-address=192.168.1.101 --bind-port=10019'

# 접속
tmux attach -t api-bridge-1

# Detach: Ctrl+B, D
```

---

## 성능 비교

### 리소스 사용량 (예상치)

| 방법 | 메모리 오버헤드 | CPU 오버헤드 | 시작 시간 |
|------|----------------|--------------|----------|
| Systemd | ~5MB | 무시 가능 | 빠름 |
| Shell Script | 0MB (nohup만) | 무시 가능 | 매우 빠름 |
| Supervisor | ~20MB | 낮음 | 중간 |
| PM2 | ~30MB | 낮음 | 중간 |
| Screen/Tmux | ~1-2MB | 무시 가능 | 빠름 |

### 신뢰성

| 방법 | 자동 복구 | 모니터링 | 로그 관리 | 점수 |
|------|----------|---------|----------|------|
| Systemd | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 5/5 |
| Shell Script | ⭐⭐ (Cron) | ⭐⭐ | ⭐⭐⭐⭐ | 3/5 |
| Supervisor | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 5/5 |
| PM2 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 5/5 |
| Screen/Tmux | ⭐ | ⭐ | ⭐⭐ | 1/5 |

---

## 실제 사용 예시

### 시나리오 1: 온프레미스 환경 (Root 권한 없음)
**선택**: Shell Script

```bash
# 디렉토리 구조
/opt/api-bridge/
├── bin/
│   └── api-bridge
├── config/
│   ├── config-1.yaml
│   ├── config-2.yaml
│   └── config-3.yaml
├── scripts/
│   ├── start.sh
│   ├── stop.sh
│   ├── restart.sh
│   ├── status.sh
│   └── watchdog.sh
├── logs/
│   ├── instance-1/
│   ├── instance-2/
│   └── instance-3/
└── pids/

# 배포 프로세스
1. 바이너리 업로드
2. ./scripts/stop.sh 1
3. ./scripts/start.sh 1
4. Health Check 확인
```

### 시나리오 2: 클라우드 환경 (Root 권한 있음)
**선택**: Systemd

```bash
# Systemd 서비스로 등록
sudo systemctl enable api-bridge-1.service
sudo systemctl start api-bridge-1.service

# 배포 프로세스
1. 새 바이너리 업로드
2. sudo systemctl restart api-bridge-1.service
3. 자동 재시작 및 Health Check
```

### 시나리오 3: 여러 마이크로서비스 통합 관리
**선택**: Supervisor

```bash
# Supervisor로 여러 서비스 통합 관리
sudo supervisorctl status
sudo supervisorctl restart api-bridge:*
sudo supervisorctl restart other-service:*

# 웹 UI로 모니터링
http://localhost:9001
```

---

## 마이그레이션 가이드

### Shell Script → Systemd

```bash
# 1. Shell Script 중지
./stop.sh 1

# 2. Systemd 서비스 파일 생성
sudo vim /etc/systemd/system/api-bridge-1.service

# 3. 서비스 등록 및 시작
sudo systemctl daemon-reload
sudo systemctl enable api-bridge-1.service
sudo systemctl start api-bridge-1.service

# 4. 확인
sudo systemctl status api-bridge-1.service
```

### Systemd → Shell Script

```bash
# 1. Systemd 서비스 중지 및 비활성화
sudo systemctl stop api-bridge-1.service
sudo systemctl disable api-bridge-1.service

# 2. Shell Script로 시작
./start.sh 1

# 3. 확인
./status.sh
```

---

## 결론

### 최종 추천

| 환경 | 1순위 | 2순위 | 3순위 |
|------|-------|-------|-------|
| **프로덕션 (Root O)** | Systemd | Supervisor | Shell Script |
| **프로덕션 (Root X)** | **Shell Script** | - | - |
| **개발/테스트** | Shell Script | Screen/Tmux | - |
| **마이크로서비스 다수** | Supervisor | Systemd | PM2 |

### API Bridge 프로젝트 선택: Shell Script

**선택 이유**:
1. ✅ 온프레미스 환경에서 Root 권한 없이 운영 가능
2. ✅ 가상IP 기반 다중 인스턴스 관리에 최적화
3. ✅ 간단한 구조로 빠른 배포 및 롤백
4. ✅ 배포 스크립트와 쉬운 통합
5. ✅ 외부 의존성 없음

**보완 방법**:
- Cron 기반 watchdog 스크립트로 자동 재시작
- Logrotate로 로그 관리
- Health Check 통합 모니터링

---

## 가상IP 설정 및 Shell Script 구현

### 가상IP 설정 방법

#### 1. 가상IP Alias 생성 (ifconfig/ip 명령어)

```bash
# 방법 1: ifconfig (전통적 방식)
sudo ifconfig eth0:0 192.168.1.101 netmask 255.255.255.0 up
sudo ifconfig eth0:1 192.168.1.102 netmask 255.255.255.0 up
sudo ifconfig eth0:2 192.168.1.103 netmask 255.255.255.0 up

# 방법 2: ip 명령어 (권장)
sudo ip addr add 192.168.1.101/24 dev eth0 label eth0:0
sudo ip addr add 192.168.1.102/24 dev eth0 label eth0:1
sudo ip addr add 192.168.1.103/24 dev eth0 label eth0:2

# 확인
ip addr show eth0
```

#### 2. 영구 설정 (재부팅 시에도 유지)

**CentOS/RHEL 7+ (nmcli 사용)**:
```bash
# Instance 1 가상IP
nmcli connection modify eth0 +ipv4.addresses 192.168.1.101/24

# Instance 2 가상IP
nmcli connection modify eth0 +ipv4.addresses 192.168.1.102/24

# Instance 3 가상IP
nmcli connection modify eth0 +ipv4.addresses 192.168.1.103/24

# 재시작
nmcli connection down eth0 && nmcli connection up eth0
```

**CentOS/RHEL 6 (네트워크 스크립트)**:
```bash
# /etc/sysconfig/network-scripts/ifcfg-eth0:0
cat > /etc/sysconfig/network-scripts/ifcfg-eth0:0 <<EOF
DEVICE=eth0:0
BOOTPROTO=static
IPADDR=192.168.1.101
NETMASK=255.255.255.0
ONBOOT=yes
EOF
```

---

### Shell Script 상세 구현

#### start.sh (인스턴스 시작 스크립트)

```bash
#!/bin/bash
# start.sh - API Bridge 인스턴스 시작 스크립트

# 설정
INSTANCE_ID=$1
BASE_DIR="/opt/api-bridge"
BINARY="${BASE_DIR}/bin/api-bridge"
CONFIG_DIR="${BASE_DIR}/config"
LOG_DIR="${BASE_DIR}/logs/instance-${INSTANCE_ID}"
PID_DIR="${BASE_DIR}/pids"

# 인스턴스별 가상IP 매핑
declare -A VIP_MAP
VIP_MAP[1]="192.168.1.101"
VIP_MAP[2]="192.168.1.102"
VIP_MAP[3]="192.168.1.103"

BIND_ADDRESS="${VIP_MAP[$INSTANCE_ID]}"
BIND_PORT="10019"
CONFIG_FILE="${CONFIG_DIR}/config-${INSTANCE_ID}.yaml"
PID_FILE="${PID_DIR}/api-bridge-${INSTANCE_ID}.pid"
LOG_FILE="${LOG_DIR}/app.log"

# 유효성 검사
if [ -z "$INSTANCE_ID" ]; then
    echo "Usage: $0 <instance_id>"
    echo "Example: $0 1"
    exit 1
fi

if [ -z "$BIND_ADDRESS" ]; then
    echo "Error: Invalid instance ID: $INSTANCE_ID"
    exit 1
fi

# 이미 실행 중인지 확인
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null 2>&1; then
        echo "Instance $INSTANCE_ID is already running (PID: $PID)"
        exit 1
    else
        echo "Removing stale PID file..."
        rm -f "$PID_FILE"
    fi
fi

# 디렉토리 생성
mkdir -p "$LOG_DIR"
mkdir -p "$PID_DIR"

# 백그라운드 실행
echo "Starting API Bridge Instance $INSTANCE_ID..."
echo "  Bind Address: $BIND_ADDRESS:$BIND_PORT"
echo "  Config File: $CONFIG_FILE"
echo "  Log File: $LOG_FILE"

nohup "$BINARY" \
    --bind-address="$BIND_ADDRESS" \
    --bind-port="$BIND_PORT" \
    --config="$CONFIG_FILE" \
    >> "$LOG_FILE" 2>&1 &

PID=$!
echo $PID > "$PID_FILE"

# 시작 확인
sleep 2
if ps -p $PID > /dev/null 2>&1; then
    echo "Instance $INSTANCE_ID started successfully (PID: $PID)"
    
    # 헬스체크
    sleep 1
    if curl -f "http://${BIND_ADDRESS}:${BIND_PORT}/health" > /dev/null 2>&1; then
        echo "Health check passed!"
    else
        echo "Warning: Health check failed"
    fi
else
    echo "Error: Failed to start instance $INSTANCE_ID"
    rm -f "$PID_FILE"
    exit 1
fi
```

#### stop.sh (인스턴스 중지 스크립트)

```bash
#!/bin/bash
# stop.sh - API Bridge 인스턴스 중지 스크립트

INSTANCE_ID=$1
BASE_DIR="/opt/api-bridge"
PID_DIR="${BASE_DIR}/pids"
PID_FILE="${PID_DIR}/api-bridge-${INSTANCE_ID}.pid"

if [ -z "$INSTANCE_ID" ]; then
    echo "Usage: $0 <instance_id>"
    exit 1
fi

if [ ! -f "$PID_FILE" ]; then
    echo "Instance $INSTANCE_ID is not running (PID file not found)"
    exit 0
fi

PID=$(cat "$PID_FILE")

if ! ps -p $PID > /dev/null 2>&1; then
    echo "Instance $INSTANCE_ID is not running (process not found)"
    rm -f "$PID_FILE"
    exit 0
fi

echo "Stopping API Bridge Instance $INSTANCE_ID (PID: $PID)..."

# Graceful shutdown (SIGTERM)
kill -TERM $PID

# 최대 30초 대기
for i in {1..30}; do
    if ! ps -p $PID > /dev/null 2>&1; then
        echo "Instance $INSTANCE_ID stopped successfully"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# 강제 종료 (SIGKILL)
echo "Forcing shutdown..."
kill -9 $PID
rm -f "$PID_FILE"
echo "Instance $INSTANCE_ID stopped (forced)"
```

#### restart.sh (재시작 스크립트)

```bash
#!/bin/bash
# restart.sh - API Bridge 인스턴스 재시작 스크립트

INSTANCE_ID=$1
BASE_DIR="/opt/api-bridge"

if [ -z "$INSTANCE_ID" ]; then
    echo "Usage: $0 <instance_id>"
    exit 1
fi

echo "Restarting API Bridge Instance $INSTANCE_ID..."

# 중지
"${BASE_DIR}/scripts/stop.sh" "$INSTANCE_ID"

# 잠시 대기
sleep 2

# 시작
"${BASE_DIR}/scripts/start.sh" "$INSTANCE_ID"
```

#### status.sh (상태 확인 스크립트)

```bash
#!/bin/bash
# status.sh - 전체 인스턴스 상태 확인

BASE_DIR="/opt/api-bridge"
PID_DIR="${BASE_DIR}/pids"

declare -A VIP_MAP
VIP_MAP[1]="192.168.1.101"
VIP_MAP[2]="192.168.1.102"
VIP_MAP[3]="192.168.1.103"

echo "API Bridge Instances Status"
echo "============================"

for INSTANCE_ID in 1 2 3; do
    PID_FILE="${PID_DIR}/api-bridge-${INSTANCE_ID}.pid"
    VIP="${VIP_MAP[$INSTANCE_ID]}"
    
    echo -n "Instance $INSTANCE_ID (${VIP}:10019): "
    
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null 2>&1; then
            # 헬스체크
            if curl -f "http://${VIP}:10019/health" > /dev/null 2>&1; then
                echo "RUNNING (PID: $PID) ✓"
            else
                echo "RUNNING (PID: $PID) - Health check FAILED ✗"
            fi
        else
            echo "STOPPED (stale PID file)"
        fi
    else
        echo "STOPPED"
    fi
done
```

#### watchdog.sh (프로세스 감시 스크립트)

```bash
#!/bin/bash
# watchdog.sh - 프로세스 감시 및 자동 재시작

BASE_DIR="/opt/api-bridge"
LOG_FILE="${BASE_DIR}/logs/watchdog.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

declare -A VIP_MAP
VIP_MAP[1]="192.168.1.101"
VIP_MAP[2]="192.168.1.102"
VIP_MAP[3]="192.168.1.103"

for INSTANCE_ID in 1 2 3; do
    VIP="${VIP_MAP[$INSTANCE_ID]}"
    
    # 헬스체크
    if ! curl -f "http://${VIP}:10019/health" > /dev/null 2>&1; then
        log "Instance $INSTANCE_ID health check failed. Restarting..."
        "${BASE_DIR}/scripts/restart.sh" "$INSTANCE_ID" >> "$LOG_FILE" 2>&1
        log "Instance $INSTANCE_ID restart completed."
    fi
done
```

**Cron 등록**:
```bash
# Cron 편집
crontab -e

# 5분마다 watchdog 실행
*/5 * * * * /opt/api-bridge/scripts/watchdog.sh
```

---

### 사용 방법

#### 초기 설정

```bash
# 1. 디렉토리 생성
mkdir -p /opt/api-bridge/{bin,config,scripts,logs,pids}

# 2. 스크립트 복사
cp start.sh stop.sh restart.sh status.sh watchdog.sh /opt/api-bridge/scripts/

# 3. 실행 권한 부여
chmod +x /opt/api-bridge/scripts/*.sh

# 4. 가상IP 설정
sudo ip addr add 192.168.1.101/24 dev eth0 label eth0:0
sudo ip addr add 192.168.1.102/24 dev eth0 label eth0:1
sudo ip addr add 192.168.1.103/24 dev eth0 label eth0:2
```

#### 일상 운영

```bash
# 인스턴스 시작
cd /opt/api-bridge/scripts
./start.sh 1
./start.sh 2
./start.sh 3

# 전체 상태 확인
./status.sh

# 특정 인스턴스 재시작
./restart.sh 2

# 특정 인스턴스 중지
./stop.sh 1

# 로그 확인
tail -f /opt/api-bridge/logs/instance-1/app.log

# Watchdog 로그 확인
tail -f /opt/api-bridge/logs/watchdog.log
```

#### 배포 프로세스

```bash
# 1. 새 바이너리 업로드
scp api-bridge user@server:/tmp/

# 2. 순차 배포 (Rolling Update)
for instance in 1 2 3; do
    echo "Deploying instance $instance..."
    
    # 인스턴스 중지
    ./stop.sh $instance
    
    # 바이너리 교체
    cp /tmp/api-bridge /opt/api-bridge/bin/
    chmod +x /opt/api-bridge/bin/api-bridge
    
    # 인스턴스 시작
    ./start.sh $instance
    
    # 헬스체크 대기
    sleep 5
    
    # 다음 인스턴스로
done

echo "Deployment completed!"
./status.sh
```

---

### 디렉토리 구조

```
/opt/api-bridge/
├── bin/
│   └── api-bridge                 # Go 바이너리
├── config/
│   ├── config-1.yaml              # Instance 1 설정
│   ├── config-2.yaml              # Instance 2 설정
│   └── config-3.yaml              # Instance 3 설정
├── scripts/
│   ├── start.sh                   # 시작 스크립트
│   ├── stop.sh                    # 중지 스크립트
│   ├── restart.sh                 # 재시작 스크립트
│   ├── status.sh                  # 상태 확인 스크립트
│   ├── watchdog.sh                # 감시 스크립트
│   └── deploy.sh                  # 배포 스크립트
├── logs/
│   ├── instance-1/
│   │   └── app.log
│   ├── instance-2/
│   │   └── app.log
│   ├── instance-3/
│   │   └── app.log
│   └── watchdog.log
└── pids/
    ├── api-bridge-1.pid
    ├── api-bridge-2.pid
    └── api-bridge-3.pid
```

---

## Go 애플리케이션 코드

### 가상IP 바인딩 구현

```go
package main

import (
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    "context"

    "github.com/gin-gonic/gin"
)

func main() {
    // 커맨드 라인 플래그
    bindAddress := flag.String("bind-address", "0.0.0.0", "Bind IP address")
    bindPort := flag.Int("bind-port", 10019, "Bind port")
    configFile := flag.String("config", "config.yaml", "Config file path")
    flag.Parse()

    // 바인딩 주소 설정
    listenAddr := fmt.Sprintf("%s:%d", *bindAddress, *bindPort)
    
    log.Printf("Starting API Bridge on %s", listenAddr)
    log.Printf("Config file: %s", *configFile)
    log.Printf("Instance VIP: %s", *bindAddress)

    // Gin 라우터 설정
    router := gin.Default()
    
    // Health Check 엔드포인트
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "healthy",
            "instance_ip": *bindAddress,
            "timestamp": time.Now(),
        })
    })
    
    // Readiness Check 엔드포인트
    router.GET("/ready", func(c *gin.Context) {
        // DB, Redis 연결 확인 로직
        c.JSON(200, gin.H{
            "status": "ready",
            "instance_ip": *bindAddress,
        })
    })

    // HTTP 서버 설정
    server := &http.Server{
        Addr:         listenAddr,
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // 서버 시작 (고루틴)
    go func() {
        log.Printf("Server listening on %s", listenAddr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start server: %v", err)
        }
    }()

    log.Printf("Server started successfully on %s", listenAddr)

    // 시그널 대기 (Graceful Shutdown)
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Graceful Shutdown (30초 타임아웃)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server forced to shutdown: %v", err)
    }

    log.Println("Server exited")
}
```

### 설정 파일 예시 (config.yaml)

```yaml
# config-1.yaml (Instance 1)
server:
  bind_address: "192.168.1.101"
  bind_port: 10019
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

legacy:
  base_url: "http://legacy-api.example.com"
  timeout: 5s

modern:
  base_url: "http://modern-api.example.com"
  timeout: 5s

database:
  driver: "oracle"
  dsn: "oracle://user:password@localhost:1521/ORCL"
  max_open_conns: 25
  max_idle_conns: 5

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

logging:
  level: "info"
  format: "json"
  output: "/opt/api-bridge/logs/instance-1/app.log"
```

---

## 배포 스크립트

### deploy.sh (Shell Script 기반 배포)

```bash
#!/bin/bash
# deploy.sh - 가상IP 기반 다중 인스턴스 배포 (Shell Script 방식)

set -e

BASE_DIR="/opt/api-bridge"
BINARY="${BASE_DIR}/bin/api-bridge"
NEW_BINARY="/tmp/api-bridge"
CONFIG_DIR="${BASE_DIR}/config"
INSTANCES=("1" "2" "3")

# 색상 코드
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 새 바이너리 확인
if [ ! -f "$NEW_BINARY" ]; then
    log_error "New binary not found: $NEW_BINARY"
    exit 1
fi

log_info "Starting deployment..."
log_info "Binary: $NEW_BINARY"

# 백업
BACKUP_DIR="${BASE_DIR}/backup/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp "$BINARY" "$BACKUP_DIR/" 2>/dev/null || log_warn "No existing binary to backup"

# 순차 배포 (Rolling Update)
for instance in "${INSTANCES[@]}"; do
    log_info "=== Deploying instance $instance ==="
    
    # 1. 인스턴스 중지
    log_info "Stopping instance $instance..."
    "${BASE_DIR}/scripts/stop.sh" "$instance"
    
    # 2. 로드밸런서 헬스체크 실패 대기
    sleep 3
    
    # 3. 바이너리 교체
    log_info "Replacing binary..."
    cp "$NEW_BINARY" "$BINARY"
    chmod +x "$BINARY"
    
    # 4. 인스턴스 시작
    log_info "Starting instance $instance..."
    "${BASE_DIR}/scripts/start.sh" "$instance"
    
    # 5. 헬스체크 대기
    declare -A VIP_MAP
    VIP_MAP[1]="192.168.1.101"
    VIP_MAP[2]="192.168.1.102"
    VIP_MAP[3]="192.168.1.103"
    
    VIP="${VIP_MAP[$instance]}"
    
    log_info "Waiting for instance $instance (${VIP}:10019) to be healthy..."
    
    RETRY_COUNT=0
    MAX_RETRIES=30
    
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -f "http://${VIP}:10019/health" > /dev/null 2>&1; then
            log_info "Instance $instance is healthy! ✓"
            break
        fi
        
        RETRY_COUNT=$((RETRY_COUNT + 1))
        
        if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
            log_error "Instance $instance health check timeout!"
            log_error "Rolling back..."
            
            # 롤백
            cp "$BACKUP_DIR/api-bridge" "$BINARY"
            "${BASE_DIR}/scripts/start.sh" "$instance"
            
            exit 1
        fi
        
        sleep 1
    done
    
    log_info "Instance $instance deployed successfully!"
    
    # 다음 인스턴스 배포 전 대기
    if [ "$instance" != "${INSTANCES[-1]}" ]; then
        log_info "Waiting before next instance..."
        sleep 5
    fi
done

log_info "==================================="
log_info "All instances deployed successfully!"
log_info "==================================="

# 최종 상태 확인
"${BASE_DIR}/scripts/status.sh"

# 백업 정리 (최근 5개만 유지)
log_info "Cleaning up old backups..."
cd "${BASE_DIR}/backup" && ls -t | tail -n +6 | xargs -r rm -rf
```

### 사용 방법

```bash
# 1. 새 바이너리를 /tmp로 업로드
scp api-bridge user@server:/tmp/

# 2. 배포 실행
cd /opt/api-bridge/scripts
./deploy.sh

# 3. 배포 확인
./status.sh
```

### 롤백 스크립트 (rollback.sh)

```bash
#!/bin/bash
# rollback.sh - 이전 버전으로 롤백

set -e

BASE_DIR="/opt/api-bridge"
BINARY="${BASE_DIR}/bin/api-bridge"
BACKUP_DIR="${BASE_DIR}/backup"
INSTANCES=("1" "2" "3")

# 최신 백업 찾기
LATEST_BACKUP=$(ls -t "$BACKUP_DIR" | head -1)

if [ -z "$LATEST_BACKUP" ]; then
    echo "No backup found!"
    exit 1
fi

BACKUP_BINARY="${BACKUP_DIR}/${LATEST_BACKUP}/api-bridge"

if [ ! -f "$BACKUP_BINARY" ]; then
    echo "Backup binary not found: $BACKUP_BINARY"
    exit 1
fi

echo "Rolling back to: $LATEST_BACKUP"

# 순차 롤백
for instance in "${INSTANCES[@]}"; do
    echo "Rolling back instance $instance..."
    
    # 중지
    "${BASE_DIR}/scripts/stop.sh" "$instance"
    
    # 바이너리 복구
    cp "$BACKUP_BINARY" "$BINARY"
    chmod +x "$BINARY"
    
    # 시작
    "${BASE_DIR}/scripts/start.sh" "$instance"
    
    # 헬스체크
    sleep 5
done

echo "Rollback completed!"
"${BASE_DIR}/scripts/status.sh"
```

---

## 고급 배포 전략

### Blue-Green 배포

```bash
#!/bin/bash
# blue-green-deploy.sh - Blue-Green 배포 전략

# 1. Green 환경에 새 버전 배포 (포트 20019)
# 2. Green 환경 헬스체크
# 3. 로드밸런서 스위칭 (Blue → Green)
# 4. Blue 환경 정리

# 구현 생략 (실제 환경에 맞게 구현)
```

### Canary 배포

```bash
#!/bin/bash
# canary-deploy.sh - Canary 배포 전략

# 1. Instance 1만 새 버전 배포 (10% 트래픽)
# 2. 모니터링 (에러율, 레이턴시)
# 3. 문제 없으면 나머지 인스턴스 배포

# Instance 1 배포
./stop.sh 1
cp /tmp/api-bridge /opt/api-bridge/bin/
./start.sh 1

# 모니터링 (10분)
echo "Monitoring canary instance for 10 minutes..."
sleep 600

# 에러율 확인 (Prometheus 쿼리 등)
# ERROR_RATE=$(curl -s 'http://prometheus:9090/api/v1/query?query=...')

# 문제 없으면 나머지 배포
echo "Canary successful. Deploying to all instances..."
for instance in 2 3; do
    ./stop.sh $instance
    ./start.sh $instance
    sleep 5
done
```

---

## 운영 팁

### 로그 로테이션 (logrotate)

```bash
# /etc/logrotate.d/api-bridge
/opt/api-bridge/logs/*/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 apibridge apibridge
    sharedscripts
    postrotate
        # HUP 시그널로 로그 파일 재오픈 (Go 애플리케이션에서 처리 필요)
        killall -HUP api-bridge 2>/dev/null || true
    endscript
}
```

### 모니터링 알림 (예시)

```bash
#!/bin/bash
# alert.sh - 모니터링 알림

# Slack Webhook
SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

send_alert() {
    local message=$1
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"🚨 API Bridge Alert: $message\"}" \
        "$SLACK_WEBHOOK"
}

# 에러율 체크
ERROR_RATE=$(curl -s 'http://prometheus:9090/api/v1/query?query=...' | jq '.data.result[0].value[1]')

if (( $(echo "$ERROR_RATE > 0.05" | bc -l) )); then
    send_alert "Error rate is high: ${ERROR_RATE}"
fi
```
