# install.ps1 - Windows installer for gitpilot
# Usage: irm https://raw.githubusercontent.com/mohammadumar-dev/gitpilot/main/install.ps1 | iex

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo    = "mohammadumar-dev/gitpilot"
$Binary  = "gitpilot"
$InstDir = "$env:LOCALAPPDATA\Programs\gitpilot"

function Write-Step  { param($msg) Write-Host "  $msg" -ForegroundColor Cyan }
function Write-Ok    { param($msg) Write-Host "  [OK] $msg" -ForegroundColor Green }
function Write-Fail  { param($msg) Write-Host "  [FAIL] $msg" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  Git Pilot Installer" -ForegroundColor White
Write-Host "  -------------------" -ForegroundColor DarkGray
Write-Host ""

# ── Detect architecture ───────────────────────────────────────────────────────
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
Write-Step "Detected architecture: windows/$arch"

# ── Fetch latest release tag ──────────────────────────────────────────────────
Write-Step "Fetching latest release tag..."
try {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $version = $release.tag_name
} catch {
    Write-Fail "Failed to fetch release info: $_"
}
Write-Ok "Latest version: $version"

# ── Build download URLs ───────────────────────────────────────────────────────
$archive      = "${Binary}_${version}_windows_${arch}.zip"
$baseUrl      = "https://github.com/$Repo/releases/download/$version"
$archiveUrl   = "$baseUrl/$archive"
$checksumUrl  = "$baseUrl/checksums.txt"

# ── Download to temp directory ────────────────────────────────────────────────
$tmpDir = Join-Path $env:TEMP "gitpilot-install"
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

$archivePath  = Join-Path $tmpDir $archive
$checksumPath = Join-Path $tmpDir "checksums.txt"

Write-Step "Downloading $archive..."
try {
    Invoke-WebRequest -Uri $archiveUrl  -OutFile $archivePath  -UseBasicParsing
    Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -UseBasicParsing
} catch {
    Write-Fail "Download failed: $_"
}
Write-Ok "Download complete"

# ── Verify SHA256 checksum ────────────────────────────────────────────────────
Write-Step "Verifying checksum..."
$expectedLine = Get-Content $checksumPath | Where-Object { $_ -match [regex]::Escape($archive) }
if (-not $expectedLine) {
    Write-Fail "Checksum entry for $archive not found in checksums.txt"
}
$expectedHash = ($expectedLine -split '\s+')[0].ToUpper()
$actualHash   = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToUpper()

if ($actualHash -ne $expectedHash) {
    Write-Fail "Checksum mismatch!`n  Expected: $expectedHash`n  Got:      $actualHash"
}
Write-Ok "Checksum verified"

# ── Extract binary ────────────────────────────────────────────────────────────
Write-Step "Extracting..."
$extractDir = Join-Path $tmpDir "extracted"
Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
Write-Ok "Extracted"

# ── Install binary ────────────────────────────────────────────────────────────
Write-Step "Installing to $InstDir..."
New-Item -ItemType Directory -Force -Path $InstDir | Out-Null
Copy-Item -Path (Join-Path $extractDir "${Binary}.exe") -Destination $InstDir -Force
Write-Ok "Binary installed"

# ── Add to user PATH if not already present ───────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstDir", "User")
    $env:Path += ";$InstDir"
    Write-Ok "Added $InstDir to user PATH"
} else {
    Write-Ok "$InstDir already in PATH"
}

# ── Cleanup ───────────────────────────────────────────────────────────────────
Remove-Item -Recurse -Force $tmpDir

# ── Verify ───────────────────────────────────────────────────────────────────
Write-Host ""
$installedBin = Join-Path $InstDir "${Binary}.exe"
& $installedBin version
Write-Host ""
Write-Host "  gitpilot $version installed successfully!" -ForegroundColor Green
Write-Host "  Restart your terminal if 'gitpilot' is not found." -ForegroundColor DarkGray
Write-Host ""
