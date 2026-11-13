# One-click build script for go-music-tools.
# Steps: precheck pnpm/go -> install frontend deps -> build frontend -> build desktop app.
# Product: build/bin/music-tools.exe (Wails branch and go build fallback use the same path).
# All comments and output are in English by design.

$ErrorActionPreference = "Stop"

$FrontendDir = "frontend"
$DistIndex = Join-Path $FrontendDir "dist/index.html"
$OutputPath = "build/bin/music-tools.exe"

function Fail($message) {
    Write-Host "[build] ERROR: $message" -ForegroundColor Red
    exit 1
}

# --- Step 1: precheck required tools -------------------------------------
Write-Host "[build] Step 1/4: prechecking required tools..."

$pnpm = Get-Command pnpm -ErrorAction SilentlyContinue
if (-not $pnpm) {
    Fail "pnpm not found in PATH. Install pnpm 9.x: npm install -g pnpm@9"
}

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    Fail "go not found in PATH. Install Go 1.26+."
}

Write-Host "[build] pnpm: $(& pnpm --version), go: $(& go version)"
Write-Host "[build] Step 1/4: precheck passed."
Write-Host ""

# --- Step 2: install frontend dependencies --------------------------------
Write-Host "[build] Step 2/4: installing frontend dependencies..."
& pnpm --dir $FrontendDir install --frozen-lockfile
if ($LASTEXITCODE -ne 0) {
    Fail "pnpm install failed with exit code $LASTEXITCODE."
}
Write-Host "[build] Step 2/4: frontend dependencies installed."
Write-Host ""

# --- Step 3: build frontend (vite build, vue-tsc type check included) -----
Write-Host "[build] Step 3/4: building frontend..."
& pnpm --dir $FrontendDir run build
if ($LASTEXITCODE -ne 0) {
    Fail "frontend build failed with exit code $LASTEXITCODE."
}
if (-not (Test-Path $DistIndex)) {
    Fail "frontend build output missing: $DistIndex"
}
Write-Host "[build] Step 3/4: frontend built and $DistIndex verified."
Write-Host ""

# --- Step 4: build desktop app --------------------------------------------
# Prefer Wails CLI when available; otherwise fall back to plain go build.
Write-Host "[build] Step 4/4: building desktop app..."

$wails = Get-Command wails -ErrorAction SilentlyContinue
if ($wails) {
    Write-Host "[build] wails CLI detected, running 'wails build'..."
    & wails build
    if ($LASTEXITCODE -ne 0) {
        Fail "wails build failed with exit code $LASTEXITCODE."
    }
} else {
    Write-Host "[build] wails CLI not found, falling back to 'go build'..."
    & go build -ldflags="-H windowsgui" -o $OutputPath .
    if ($LASTEXITCODE -ne 0) {
        Fail "go build failed with exit code $LASTEXITCODE."
    }
}

if (-not (Test-Path $OutputPath)) {
    Fail "build output missing: $OutputPath"
}

Write-Host ""
Write-Host "[build] SUCCESS. Output: $OutputPath" -ForegroundColor Green
