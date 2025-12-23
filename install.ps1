# EZ Language Installer for Windows
#
# Copyright (c) 2025-Present Marshall A Burns
# Licensed under the MIT License. See LICENSE for details.
#
# Run as Administrator for system-wide installation,
# or as regular user for user-local installation.

$ErrorActionPreference = "Stop"

Write-Host "EZ Language Installer" -ForegroundColor Cyan
Write-Host "====================" -ForegroundColor Cyan
Write-Host ""

# Check if running as Administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if ($isAdmin) {
    $InstallDir = "$env:ProgramFiles\ez"
    $PathScope = "Machine"
    Write-Host "Running as Administrator - installing system-wide" -ForegroundColor Green
} else {
    $InstallDir = "$env:LOCALAPPDATA\ez"
    $PathScope = "User"
    Write-Host "Running as regular user - installing for current user only" -ForegroundColor Yellow
    Write-Host "(Run as Administrator for system-wide installation)" -ForegroundColor Gray
}

Write-Host ""

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "Found: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Go is not installed" -ForegroundColor Red
    Write-Host "Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Build the binary
Write-Host ""
Write-Host "Building EZ..." -ForegroundColor Cyan
try {
    go build -o ez.exe ./cmd/ez
    if (-not (Test-Path "ez.exe")) {
        throw "Build failed - ez.exe not created"
    }
    Write-Host "Build successful" -ForegroundColor Green
} catch {
    Write-Host "Error: Build failed" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    exit 1
}

# Create install directory
Write-Host ""
Write-Host "Installing to $InstallDir..." -ForegroundColor Cyan
try {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Write-Host "Created directory: $InstallDir" -ForegroundColor Green
    }

    # Move binary to install directory
    Move-Item -Path "ez.exe" -Destination "$InstallDir\ez.exe" -Force
    Write-Host "Installed ez.exe to $InstallDir" -ForegroundColor Green
} catch {
    Write-Host "Error: Failed to install" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    # Clean up local binary if it exists
    if (Test-Path "ez.exe") { Remove-Item "ez.exe" -Force }
    exit 1
}

# Add to PATH if not already present
$currentPath = [Environment]::GetEnvironmentVariable("Path", $PathScope)
if ($currentPath -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "Adding $InstallDir to PATH..." -ForegroundColor Cyan
    try {
        $newPath = "$currentPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, $PathScope)
        Write-Host "Added to $PathScope PATH" -ForegroundColor Green
        Write-Host "Note: Restart your terminal for PATH changes to take effect" -ForegroundColor Yellow
    } catch {
        Write-Host "Warning: Could not add to PATH automatically" -ForegroundColor Yellow
        Write-Host "Please add '$InstallDir' to your PATH manually" -ForegroundColor Yellow
    }
} else {
    Write-Host "$InstallDir is already in PATH" -ForegroundColor Green
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "EZ installed successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Run 'ez <file>' to execute EZ programs" -ForegroundColor White
Write-Host ""
Write-Host "To uninstall, run:" -ForegroundColor Gray
Write-Host "  Remove-Item -Recurse -Force '$InstallDir'" -ForegroundColor Gray
if ($PathScope -eq "Machine") {
    Write-Host "  (Run as Administrator)" -ForegroundColor Gray
}
Write-Host ""
