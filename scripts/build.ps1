#!/usr/bin/env pwsh
#
# ChronoGo Build Script for PowerShell
#

param (
    [switch]$Release,
    [switch]$Test,
    [switch]$Clean,
    [switch]$Lint,
    [switch]$All,
    [switch]$Help,
    [string]$Output = "chrono.exe"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = (Get-Item $PSScriptRoot).Parent.FullName
$BuildTime = Get-Date -UFormat "%Y-%m-%dT%H:%M:%SZ"
$Version = "0.1.0" # TODO: Use git tags for versioning

# Show help and exit
function ShowHelp {
    Write-Host "ChronoGo Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\scripts\build.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Release     Build in release mode (optimized, stripped)"
    Write-Host "  -Test        Run tests after building"
    Write-Host "  -Clean       Clean build artifacts"
    Write-Host "  -Lint        Run linter"
    Write-Host "  -All         Clean, lint, build, and test"
    Write-Host "  -Output      Specify output filename (default: chrono.exe)"
    Write-Host "  -Help        Show this help message"
    Write-Host ""
    Write-Host "Example: .\scripts\build.ps1 -Release -Test"
    exit 0
}

# Check for help flag
if ($Help) {
    ShowHelp
}

# Build flags
$CommonFlags = @(
    "-trimpath"
)

if ($Release) {
    $BuildFlags = $CommonFlags + @(
        "-ldflags", "-s -w -X 'github.com/willibrandon/ChronoGo/pkg/version.Version=$Version' -X 'github.com/willibrandon/ChronoGo/pkg/version.BuildTime=$BuildTime'"
    )
    Write-Host "Building in RELEASE mode..." -ForegroundColor Green
} else {
    $BuildFlags = $CommonFlags + @(
        "-gcflags", "all=-N -l", # Disable optimizations for debugging
        "-ldflags", "-X 'github.com/willibrandon/ChronoGo/pkg/version.Version=dev-$Version' -X 'github.com/willibrandon/ChronoGo/pkg/version.BuildTime=$BuildTime'"
    )
    Write-Host "Building in DEBUG mode..." -ForegroundColor Yellow
}

function Lint {
    Write-Host "Running linters..." -ForegroundColor Cyan
    
    # Check if golangci-lint is installed
    $golangciLint = Get-Command golangci-lint -ErrorAction SilentlyContinue
    if ($null -eq $golangciLint) {
        Write-Host "golangci-lint not found. Please install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" -ForegroundColor Red
        return 1
    }
    
    # Run linter and capture output
    $lintOutput = & golangci-lint run ./... 2>&1
    $lintExitCode = $LASTEXITCODE
    
    if ($lintExitCode -ne 0) {
        Write-Host "Linting failed with the following errors:" -ForegroundColor Red
        foreach ($line in $lintOutput) {
            Write-Host $line -ForegroundColor Yellow
        }
        return $lintExitCode
    }
    
    Write-Host "Linting passed" -ForegroundColor Green
    return 0
}

function BuildMain {
    Write-Host "Building ChronoGo main executable..." -ForegroundColor Cyan
    
    Push-Location $ProjectRoot
    try {
        & go build $BuildFlags -o $Output ./cmd/chrono
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Build failed" -ForegroundColor Red
            return $LASTEXITCODE
        }
    } finally {
        Pop-Location
    }
    
    Write-Host "Build successful: $Output" -ForegroundColor Green
    return 0
}

function RunTests {
    Write-Host "Running tests..." -ForegroundColor Cyan
    
    Push-Location $ProjectRoot
    try {
        # Run tests with verbose output and formatting
        Write-Host "Test Results:" -ForegroundColor Yellow
        Write-Host "============================================================" -ForegroundColor Yellow
        
        # Run the go test command with verbose output, passing through all output directly
        & go test -v ./... | ForEach-Object { 
            # Colorize test output based on content
            if ($_ -match "^--- PASS") {
                Write-Host $_ -ForegroundColor Green
            }
            elseif ($_ -match "^--- FAIL") {
                Write-Host $_ -ForegroundColor Red 
            }
            elseif ($_ -match "^PASS") {
                Write-Host $_ -ForegroundColor Green
            }
            elseif ($_ -match "^FAIL") {
                Write-Host $_ -ForegroundColor Red
            }
            elseif ($_ -match "^\s+Error") {
                Write-Host $_ -ForegroundColor Red
            }
            else {
                Write-Host $_
            }
        }
        
        Write-Host "============================================================" -ForegroundColor Yellow
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Tests failed" -ForegroundColor Red
            return $LASTEXITCODE
        }
    } finally {
        Pop-Location
    }
    
    Write-Host "All tests passed" -ForegroundColor Green
    return 0
}

function CleanBuild {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Cyan
    
    if (Test-Path "$ProjectRoot/$Output") {
        Remove-Item -Force "$ProjectRoot/$Output"
        Write-Host "Removed $Output" -ForegroundColor Gray
    }
    
    # Clean go test cache
    & go clean -testcache
    Write-Host "Cleaned Go test cache" -ForegroundColor Gray
    
    Write-Host "Clean complete" -ForegroundColor Green
    return 0
}

# Main script execution
if ($Clean -or $All) {
    CleanBuild
    if ($Clean -and -not $All) { exit 0 }
}

if ($Lint -or $All) {
    $lintResult = Lint
    if ($lintResult -ne 0) { exit $lintResult }
    if ($Lint -and -not $All) { exit 0 }
}

$buildResult = BuildMain
if ($buildResult -ne 0) { exit $buildResult }

if ($Test -or $All) {
    $testResult = RunTests
    if ($testResult -ne 0) { exit $testResult }
}

Write-Host "All operations completed successfully" -ForegroundColor Green 