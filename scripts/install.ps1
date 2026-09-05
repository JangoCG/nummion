# Nummion release installer for Windows PowerShell 5.1 and PowerShell 7+.
$ErrorActionPreference = 'Stop'
$repo = 'JangoCG/nummion'
$version = $env:NUMMION_VERSION
if (-not $version) {
    $release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
    $version = $release.tag_name
}
$version = $version -replace '^v', ''
if ($version -notmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$') {
    throw 'Invalid release version.'
}
$architecture = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
$arch = switch ($architecture) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw 'Supported architectures are amd64 and arm64.' }
}
$archive = "nummion_${version}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/v$version"
$work = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
$staged = $null
New-Item -ItemType Directory -Path $work | Out-Null
try {
    $zipPath = Join-Path $work $archive
    $checksums = Join-Path $work 'checksums.txt'
    Invoke-WebRequest "$base/$archive" -OutFile $zipPath -UseBasicParsing
    Invoke-WebRequest "$base/checksums.txt" -OutFile $checksums -UseBasicParsing
    if (Get-Command cosign -ErrorAction SilentlyContinue) {
        $bundle = Join-Path $work 'checksums.txt.bundle'
        Invoke-WebRequest "$base/checksums.txt.bundle" -OutFile $bundle -UseBasicParsing
        & cosign verify-blob --bundle $bundle `
            --certificate-identity "https://github.com/$repo/.github/workflows/release.yml@refs/tags/v$version" `
            --certificate-oidc-issuer https://token.actions.githubusercontent.com $checksums
        if ($LASTEXITCODE -ne 0) { throw 'Signature verification failed (cosign v3+ is required).' }
    }
    $entries = @(Get-Content $checksums | Where-Object { $_ -match ('^[0-9a-fA-F]{64}\s+' + [regex]::Escape($archive) + '$') })
    if ($entries.Count -ne 1) { throw 'Missing, duplicate, or invalid SHA-256 checksum.' }
    $expected = ($entries[0] -split '\s+')[0]
    if ((Get-FileHash $zipPath -Algorithm SHA256).Hash -ne $expected) { throw 'SHA-256 mismatch; existing installation was left untouched.' }

    # Extract only the executable; never unpack arbitrary paths from an archive.
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead($zipPath)
    $downloaded = Join-Path $work 'num.exe'
    try {
        $binaries = @($zip.Entries | Where-Object { $_.FullName -ceq 'num.exe' })
        if ($binaries.Count -ne 1) { throw 'Archive must contain exactly one num.exe.' }
        [IO.Compression.ZipFileExtensions]::ExtractToFile($binaries[0], $downloaded)
    } finally { $zip.Dispose() }
    $reported = & $downloaded --version
    if ($LASTEXITCODE -ne 0 -or $reported -cne "num $version") { throw 'Downloaded executable reports an unexpected version.' }

    $binDir = if ($env:NUMMION_BIN_DIR) { $env:NUMMION_BIN_DIR } else { Join-Path $env:LOCALAPPDATA 'Nummion\bin' }
    New-Item -ItemType Directory -Force -Path $binDir | Out-Null
    $target = Join-Path $binDir 'num.exe'
    $staged = Join-Path $binDir ('.num-install-' + [guid]::NewGuid().ToString() + '.exe')
    Copy-Item -LiteralPath $downloaded -Destination $staged
    if (Test-Path -LiteralPath $target) {
        # A PowerShell $null binds to an empty string for this .NET overload.
        [IO.File]::Replace($staged, $target, [System.Management.Automation.Language.NullString]::Value)
    } else {
        [IO.File]::Move($staged, $target)
    }
    $staged = $null
    Write-Host "Installed num $version to $target"
    Write-Host "Add this directory to your user PATH if needed: $binDir"
    Write-Host 'Next: num auth set'
    Write-Host 'Optional agent integration: num skill install'
} finally {
    Remove-Item -LiteralPath $work -Recurse -Force
    if ($staged -and (Test-Path -LiteralPath $staged)) { Remove-Item -LiteralPath $staged -Force }
}
