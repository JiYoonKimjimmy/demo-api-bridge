#!/bin/bash

# API Bridge 빌드 스크립트 (Bash)

echo "Building API Bridge..."

# 빌드 디렉토리 생성
if [ ! -d "bin" ]; then
    mkdir -p bin
fi

# 빌드 실행
OUTPUT="bin/api-bridge"
go build -ldflags="-s -w" -o $OUTPUT cmd/api-bridge/main.go

if [ $? -eq 0 ]; then
    echo "Build successful: $OUTPUT"
    
    # 파일 크기 확인
    SIZE=$(du -h $OUTPUT | cut -f1)
    echo "Binary size: $SIZE"
else
    echo "Build failed!"
    exit 1
fi
