# EZ Integration Test Runner (PowerShell)
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.

$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$PROJECT_ROOT = Split-Path -Parent $SCRIPT_DIR
$EZ_BIN = Join-Path $PROJECT_ROOT "ez.exe"

# Check if ez binary exists
if (-not (Test-Path $EZ_BIN)) {
    Write-Host "EZ binary not found, building..."
    Set-Location $PROJECT_ROOT
    go build -o ez.exe ./cmd/ez
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Failed to build EZ binary" -ForegroundColor Red
        exit 1
    }
}

$PASS_COUNT = 0
$FAIL_COUNT = 0

Write-Host "========================================"
Write-Host "  EZ Integration Test Suite"
Write-Host "========================================"
Write-Host ""

# Run fail tests (should fail)
Write-Host "Running FAIL tests (expecting errors)..."
Write-Host "----------------------------------------"

$errorTests = Get-ChildItem -Path (Join-Path $SCRIPT_DIR "fail\errors\*.ez")
foreach ($test in $errorTests) {
    $testName = $test.BaseName
    Write-Host -NoNewline "  errors/$testName... "
    
    $output = & $EZ_BIN $test.FullName 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "FAIL (expected error, got success)" -ForegroundColor Red
        $FAIL_COUNT++
    } else {
        Write-Host "PASS" -ForegroundColor Green
        $PASS_COUNT++
    }
}

# Multi-file error tests (single files)
$multiFileTests = Get-ChildItem -Path (Join-Path $SCRIPT_DIR "fail\multi-file\*.ez") -ErrorAction SilentlyContinue
foreach ($test in $multiFileTests) {
    $testName = $test.BaseName
    Write-Host -NoNewline "  multi-file/$testName... "
    
    $output = & $EZ_BIN $test.FullName 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "FAIL (expected error, got success)" -ForegroundColor Red
        $FAIL_COUNT++
    } else {
        Write-Host "PASS" -ForegroundColor Green
        $PASS_COUNT++
    }
}

# Multi-file error tests (directories with main.ez)
$multiFileDirs = Get-ChildItem -Path (Join-Path $SCRIPT_DIR "fail\multi-file") -Directory
foreach ($dir in $multiFileDirs) {
    $mainFile = Get-ChildItem -Path $dir.FullName -Recurse -Filter "main.ez" | Select-Object -First 1
    if ($mainFile) {
        $dirName = $dir.Name
        Write-Host -NoNewline "  multi-file/$dirName... "
        
        $output = & $EZ_BIN $mainFile.FullName 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "FAIL (expected error, got success)" -ForegroundColor Red
            $FAIL_COUNT++
        } else {
            Write-Host "PASS" -ForegroundColor Green
            $PASS_COUNT++
        }
    }
}

# Summary
Write-Host ""
Write-Host "========================================"
Write-Host "  Test Summary"
Write-Host "========================================"
Write-Host ""
Write-Host "  Passed:  $PASS_COUNT" -ForegroundColor Green
Write-Host "  Failed:  $FAIL_COUNT" -ForegroundColor Red
Write-Host ""

if ($FAIL_COUNT -eq 0) {
    Write-Host "  ALL TESTS PASSED!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "  SOME TESTS FAILED" -ForegroundColor Red
    exit 1
}

