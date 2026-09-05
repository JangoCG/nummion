# Offline integration checks using the Windows binary built by CI.
$ErrorActionPreference = 'Stop'
$installer = Join-Path $PSScriptRoot 'install.ps1'
$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('nummion-test-' + [guid]::NewGuid())
$fixtureDir = Join-Path $testRoot 'fixtures'
$binDir = Join-Path $testRoot 'bin with spaces'
$originalVersion = $env:NUMMION_VERSION
$originalBinDir = $env:NUMMION_BIN_DIR
New-Item -ItemType Directory -Path $fixtureDir, $binDir | Out-Null
try {
    $env:NUMMION_VERSION = '0.1.0-dev'
    $env:NUMMION_BIN_DIR = $binDir
    $fixtureBinary = Join-Path $fixtureDir 'num.exe'
    Copy-Item (Join-Path $PSScriptRoot '../bin/num') $fixtureBinary
    $archiveName = 'nummion_0.1.0-dev_windows_amd64.zip'
    $archive = Join-Path $fixtureDir $archiveName
    Compress-Archive -Path $fixtureBinary -DestinationPath $archive
    $checksumFile = Join-Path $fixtureDir 'checksums.txt'
    $checksum = (Get-FileHash $archive -Algorithm SHA256).Hash.ToLower()
    Set-Content $checksumFile "$checksum  $archiveName"
    # Scope the test double to this script; no network calls or user PATH edits.
    function Invoke-WebRequest($Uri, $OutFile, [switch]$UseBasicParsing) {
        Copy-Item (Join-Path $fixtureDir ($Uri.Split('/')[-1])) $OutFile
    }
    & $installer
    $installed = Join-Path $binDir 'num.exe'
    if ((& $installed --version) -ne 'num 0.1.0-dev') { throw 'Fresh install failed.' }
    & $installer
    if ((& $installed --version) -ne 'num 0.1.0-dev') { throw 'Upgrade failed.' }
    $before = (Get-FileHash $installed).Hash
    foreach ($invalid in @(('0' * 64 + "  $archiveName"), '', "$checksum  $archiveName`n$checksum  $archiveName")) {
        Set-Content $checksumFile $invalid
        $rejected = $false
        try { & $installer } catch { $rejected = $true }
        if (-not $rejected) { throw 'Invalid checksum was accepted.' }
        if ((Get-FileHash $installed).Hash -ne $before) { throw 'Failed install changed the existing executable.' }
    }
    Write-Host 'PowerShell installer: fresh install, upgrade, corrupt/missing/duplicate checksum checks passed.'
} finally {
    $env:NUMMION_VERSION = $originalVersion
    $env:NUMMION_BIN_DIR = $originalBinDir
    Remove-Item $testRoot -Recurse -Force
}
