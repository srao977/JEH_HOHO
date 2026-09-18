# File: Build-Product.ps1
# Date: 2026-09-17
# Version/Status: 1.0 / Slice 7 implementation
# Product/Component: JEH-HOHO / Deterministic product packaging
# Purpose: Build and validate one self-contained end-user-testable product candidate.
# Responsibilities: Generate contracts, test, compile, collect runtime assets, and hash files.
# Inputs/Outputs: Product repository in; dist/JEH-HOHO package directory out.
# Configuration: Optional output path must remain under the repository dist directory.
# Dependencies: Build-time Go, Buf, Node.js, and npm; none are required from the operator.
# Invariants: One proto generation operation; Viewer standalone static assets are included.
# Failure behavior: Any failed command stops packaging and leaves no successful manifest.
# Non-responsibilities: Credentials, installation, signing, or product freeze.

[CmdletBinding()]
param([string]$OutputDirectory)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$distRoot = Join-Path $root 'dist'
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $distRoot 'JEH-HOHO'
}
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
$allowedPrefix = [IO.Path]::GetFullPath($distRoot) + [IO.Path]::DirectorySeparatorChar
if (-not $OutputDirectory.StartsWith($allowedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Package output must be inside the repository dist directory.'
}

function Invoke-Checked([scriptblock]$Command, [string]$Name) {
    & $Command
    if ($LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE." }
}

Push-Location $root
try {
    Invoke-Checked { buf generate } 'buf generate'
    Invoke-Checked { buf lint } 'buf lint'
    Invoke-Checked { buf build } 'buf build'
    Invoke-Checked { go test ./... } 'Go tests'
    Invoke-Checked { go vet ./... } 'Go vet'
    Push-Location (Join-Path $root 'viewer')
    try {
        Invoke-Checked { npm ci } 'Viewer dependency restore'
        Invoke-Checked { npm test } 'Viewer tests'
        Invoke-Checked { npx tsc --noEmit } 'Viewer TypeScript compilation'
        Invoke-Checked { npm run lint } 'Viewer lint'
        Invoke-Checked { npm run build } 'Viewer production build'
    } finally { Pop-Location }

    if (Test-Path $OutputDirectory) { Remove-Item -LiteralPath $OutputDirectory -Recurse -Force }
    $directories = @('bin', 'config', 'contract', 'docs', 'runtime', 'viewer')
    foreach ($directory in $directories) { New-Item -ItemType Directory -Force -Path (Join-Path $OutputDirectory $directory) | Out-Null }

    Invoke-Checked { go build -trimpath -buildvcs=false -ldflags '-s -w' -o (Join-Path $OutputDirectory 'bin\jeh-hoho.exe') ./cmd/jeh-hoho } 'Controller build'

    $viewerSource = Join-Path $root 'viewer\.next\standalone'
    Get-ChildItem -LiteralPath $viewerSource -Force | Copy-Item -Destination (Join-Path $OutputDirectory 'viewer') -Recurse -Force
    if (-not (Test-Path (Join-Path $OutputDirectory 'viewer\.next\static'))) { throw 'Packaged Viewer static assets are missing.' }

    $node = (Get-Command node.exe -ErrorAction Stop).Source
    Copy-Item -LiteralPath $node -Destination (Join-Path $OutputDirectory 'runtime\node.exe')
    Copy-Item -LiteralPath (Join-Path $root 'config\product.json') -Destination (Join-Path $OutputDirectory 'config\product.json')
    Copy-Item -LiteralPath (Join-Path $root 'config\credentials.example.json') -Destination (Join-Path $OutputDirectory 'config\credentials.example.json')
    Copy-Item -LiteralPath (Join-Path $root 'api\proto\jeh_hoho\v1\JEH_HOHO.proto') -Destination (Join-Path $OutputDirectory 'contract\JEH_HOHO.proto')
    Copy-Item -LiteralPath (Join-Path $root 'buf.yaml') -Destination (Join-Path $OutputDirectory 'contract\buf.yaml')
    Copy-Item -LiteralPath (Join-Path $root 'buf.gen.yaml') -Destination (Join-Path $OutputDirectory 'contract\buf.gen.yaml')
    Copy-Item -LiteralPath (Join-Path $root 'viewer\package-lock.json') -Destination (Join-Path $OutputDirectory 'contract\viewer-package-lock.json')
    Copy-Item -LiteralPath (Join-Path $root 'docs\operator\JEH_HOHO_CHILDS_PLAY_USER_GUIDE_V1_0_2026-09-17.md') -Destination (Join-Path $OutputDirectory 'docs\USER_GUIDE.md')
    Get-ChildItem -LiteralPath (Join-Path $PSScriptRoot 'operator') -File | Copy-Item -Destination $OutputDirectory

    $files = Get-ChildItem -LiteralPath $OutputDirectory -File -Recurse | Sort-Object FullName
    $checksums = @($files | ForEach-Object {
        [ordered]@{
            path = $_.FullName.Substring($OutputDirectory.Length + 1).Replace('\', '/')
            sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
            bytes = $_.Length
        }
    })
    $manifest = [ordered]@{
        product = 'JEH-HOHO'
        version = '0.1.0-candidate'
        status = 'END-USER-TESTABLE PRODUCT CANDIDATE'
        contract_source = 'contract/JEH_HOHO.proto'
        generation = 'buf generate'
        go_version = ((go version) -join '')
        node_version = ((node --version) -join '')
        buf_version = ((buf --version) -join '')
        files = $checksums
    }
    $manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $OutputDirectory 'package-manifest.json') -Encoding UTF8

    $verification = Get-Content -LiteralPath (Join-Path $OutputDirectory 'package-manifest.json') -Raw | ConvertFrom-Json
    foreach ($entry in $verification.files) {
        $path = Join-Path $OutputDirectory ([string]$entry.path).Replace('/', '\')
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Package verification missing $($entry.path)." }
        $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $entry.sha256) { throw "Package verification failed for $($entry.path)." }
    }
    & (Join-Path $PSScriptRoot 'Test-ProductCandidate.ps1') -PackageDirectory $OutputDirectory
    Write-Host "JEH-HOHO product candidate created: $OutputDirectory"
} finally { Pop-Location }