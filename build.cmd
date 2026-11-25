@echo off
chcp 65001 >nul
echo [INFO] Building Vue 3 frontend...
cd frontend
call npm run build
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%
cd ..
if not exist fnpack\app\bin mkdir fnpack\app\bin
echo [INFO] Cross-compiling Linux x86_64 binary...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o fnpack\app\bin\antigravity.harness .
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%
echo [OK] Build complete: fnpack\app\bin\antigravity.harness
