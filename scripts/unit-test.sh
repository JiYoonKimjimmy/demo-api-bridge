#!/bin/bash

# API Bridge Unit Test Script (Bash)
# Runs Go unit tests with coverage analysis

echo "Running unit tests..."

# 테스트 실행
go test -v -race -coverprofile=coverage.out ./...

if [ $? -eq 0 ]; then
    echo -e "\nUnit tests passed!"
    
    # 커버리지 확인
    echo -e "\nGenerating coverage report..."
    go tool cover -func=coverage.out
else
    echo -e "\nUnit tests failed!"
    exit 1
fi
