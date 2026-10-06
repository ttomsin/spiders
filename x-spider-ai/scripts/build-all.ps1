# Local multi-platform build script for xsai
param (
    [string]$Version = "v1.0.0"
)

$ErrorActionPreference = "Stop"
$DistDir = "dist"
$Commit = (git rev-parse --short HEAD 2>$null)
if (-not $Commit) { $Commit = "dev" }
$BuildDate = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

if (-not (Test-Path $DistDir)) {
    New-Item -ItemType Directory -Path $DistDir | Out-Null
}

$Targets = @(
    @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" },
    @{ OS = "windows"; Arch = "arm64"; Ext = ".exe" },
    @{ OS = "linux";   Arch = "amd64"; Ext = "" },
    @{ OS = "linux";   Arch = "arm64"; Ext = "" },
    @{ OS = "darwin";  Arch = "amd64"; Ext = "" },
    @{ OS = "darwin";  Arch = "arm64"; Ext = "" }
)

Write-Host "🚀 Building xsai $Version ($Commit) across all targets..." -ForegroundColor Cyan

foreach ($t in $Targets) {
    $outName = "xsai-$($t.OS)-$($t.Arch)$($t.Ext)"
    $outPath = Join-Path $DistDir $outName

    Write-Host "  -> Building $outName..." -ForegroundColor Gray
    $env:GOOS = $t.OS
    $env:GOARCH = $t.Arch
    $env:CGO_ENABLED = "0"

    $ldflags = "-s -w -X main.version=$Version -X main.commit=$Commit -X main.date=$BuildDate"
    go build -ldflags $ldflags -o $outPath .\cmd\x-spider-ai

    # Generate SHA-256 hash
    $hash = (Get-FileHash -Path $outPath -Algorithm SHA256).Hash.ToLower()
    Set-Content -Path "$outPath.sha256" -Value "$hash  $outName"
}

# Also generate a canonical Windows binary in the root directory
Write-Host "  -> Building canonical xsai.exe and x-spider-ai.exe in root..." -ForegroundColor Gray
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags "-s -w -X main.version=$Version -X main.commit=$Commit -X main.date=$BuildDate" -o xsai.exe .\cmd\x-spider-ai
Copy-Item xsai.exe x-spider-ai.exe -Force

Write-Host "✅ All targets compiled successfully into ./$DistDir/!" -ForegroundColor Green
Get-ChildItem $DistDir | Select-Object Name, Length
