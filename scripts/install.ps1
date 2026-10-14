# ==============================================================================
# 🌼 Daisy Code Editor — One-Line Windows PowerShell Installer
#
# Usage:
#   irm https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.ps1 | iex
#
# Parameters:
#   $env:VERSION      - Specific version to install (default: latest, e.g. "v0.2.0")
#   $env:INSTALL_DIR  - Custom install directory (default: "$HOME\.daisy\bin")
# ==============================================================================

$ErrorActionPreference = "Stop"

$Repo = "raillen/daisy-code-editor"
$DefaultVersion = "v0.2.0"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$HOME\.daisy\bin" }

function Write-Banner {
    Write-Host ""
    Write-Host "  ╭────────────────────────────────────────────────────────────╮" -ForegroundColor Yellow
    Write-Host "  │               🌼 Daisy Code Editor Installer               │" -ForegroundColor Yellow
    Write-Host "  │   Modern Terminal Code Editor · OpenTUI + Solid (Bun)      │" -ForegroundColor Yellow
    Write-Host "  ╰────────────────────────────────────────────────────────────╯" -ForegroundColor Yellow
    Write-Host ""
}

function Resolve-Version {
    if ($env:VERSION) {
        return $env:VERSION
    }
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing -TimeoutSec 5
        if ($release.tag_name) {
            return $release.tag_name
        }
    } catch {
        # Fallback to default version if offline
    }
    return $DefaultVersion
}

function Add-ToPath ($Directory) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -split ";" -contains $Directory) {
        return
    }

    Write-Host "==> Adding '$Directory' to User PATH..." -ForegroundColor Cyan
    $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $Directory } else { "$userPath;$Directory" }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$Directory;$env:Path"
    Write-Host "==> User PATH successfully updated." -ForegroundColor Green
}

function Main {
    Write-Banner

    $version = Resolve-Version
    if (-not $version.StartsWith("v")) {
        $version = "v$version"
    }

    Write-Host "==> Selected version: $version" -ForegroundColor Cyan
    Write-Host "==> Install destination: $InstallDir" -ForegroundColor Cyan

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    Write-Host ""
    Write-Host "==> Daisy Code Editor is built with Bun and OpenTUI." -ForegroundColor Yellow
    Write-Host "==> To run on Windows, you can also use WSL (Ubuntu/Debian) or install Bun (https://bun.sh)." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  Getting Started:" -ForegroundColor Cyan
    Write-Host "    daisy-code ."
    Write-Host "    daisy-code --help"
    Write-Host ""
}

Main
