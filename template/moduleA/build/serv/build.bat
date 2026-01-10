@echo off
chcp 65001 > nul
setlocal enabledelayedexpansion

REM ========== 配置区域 ==========
SET OUTPUT_NAME=moduleA.exe
SET BUILD_FLAGS=-buildmode=c-shared -ldflags="-s -w"
REM =============================

echo =============================================
echo           开始编译 Go EXE 模块
echo =============================================
echo.

echo [1/4] 正在清理和整理 Go 模块依赖...
go mod tidy
if %errorlevel% neq 0 (
    echo [错误] go mod tidy 执行失败!
    pause
    exit /b %errorlevel%
)
echo [完成] 模块依赖整理完成
echo.

echo [2/4] 正在检查代码语法...
go vet ./...
if %errorlevel% neq 0 (
    echo [警告] 代码检查发现问题，但继续编译...
) else (
    echo [完成] 代码检查通过
)
echo.

echo [3/4] 正在编译 EXE 文件...
echo   目标文件: %OUTPUT_NAME%
echo   编译参数: %BUILD_FLAGS%
echo.

go build %BUILD_FLAGS% -o "%OUTPUT_NAME%" .
if %errorlevel% neq 0 (
    echo [错误] 编译失败!
    pause
    exit /b %errorlevel%
)
echo [完成] 编译成功!
echo.

echo [4/4] 输出文件信息...
echo   文件位置: %cd%\%OUTPUT_NAME%
for %%F in ("%OUTPUT_NAME%") do echo   文件大小: %%~zF 字节
echo.

echo =============================================
echo   编译完成! EXE 文件: %OUTPUT_NAME%
echo =============================================

REM 可选: 显示编译平台信息
echo.
echo 平台信息:
go version
echo.

REM 可选: 显示编译时间
echo 编译时间: %date% %time%

pause
