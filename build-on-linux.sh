#!/bin/bash
# 创建输出目录
mkdir -p build

echo "正在编译多个平台版本..."

# Linux
echo "编译 Linux 64位..."
GOOS=linux GOARCH=amd64 go build -o build/apkcheckpack_linux_amd64 ./src

echo "编译 Linux 32位..."
GOOS=linux GOARCH=386 go build -o build/apkcheckpack_linux_386 ./src

echo "编译 Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o build/apkcheckpack_linux_arm64 ./src

# Windows
echo "编译 Windows 64位..."
GOOS=windows GOARCH=amd64 go build -o build/apkcheckpack_windows_amd64.exe ./src

echo "编译 Windows 32位..."
GOOS=windows GOARCH=386 go build -o build/apkcheckpack_windows_386.exe ./src

# macOS
echo "编译 macOS Intel..."
GOOS=darwin GOARCH=amd64 go build -o build/apkcheckpack_darwin_amd64 ./src

echo "编译 macOS Apple Silicon..."
GOOS=darwin GOARCH=arm64 go build -o build/apkcheckpack_darwin_arm64 ./src

echo "编译完成！所有文件在 build/ 目录中"
# ls -la build/

# go build -o apkcheckpack ./src