# File: Test-ProductCandidate.ps1
# Date: 2026-09-17
# Version/Status: 1.0 / Slice 7 acceptance authority
# Product/Component: JEH-HOHO / Clean-package validation
# Purpose: Validate package integrity, source-tree independence, and packaged operator workflow.
# Responsibilities: Copy the candidate to isolation, verify it, and exercise packaged commands.
# Inputs/Outputs: Built candidate in; validation result and bounded process output out.
# Configuration: Full workflow uses credentials.json and approved external persistence.
# Dependencies: Windows PowerShell and the packaged product only.
# Invariants: Validation never starts binaries from the source repository or reference trees.
# Failure behavior: Any integrity, dependency, lifecycle, or cleanup failure stops acceptance.
# Non-responsibilities: Building the candidate, provisioning MongoDB, or human usability approval.

[CmdletBinding()]
param(
    [string]$PackageDirectory = (Join-Path (Split-Path -Parent $PSScriptRoot) 'dist\JEH-HOHO'),
    [switch]$FullWorkflow
)

$ErrorActionPreference = 'Stop'
$PackageDirectory = [IO.Path]::GetFullPath($PackageDirectory)
$manifestPath = Join-Path $PackageDirectory 'package-manifest.json'
if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "Package manifest not found: $manifestPath"
}

$isolationRoot = Join-Path ([IO.Path]::GetTempPath()) ("JEH-HOHO-Acceptance-" + [guid]::NewGuid().ToString('N'))
$isolatedPackage = Join-Path $isolationRoot 'JEH-HOHO'
New-Item -ItemType Directory -Force -Path $isolatedPackage | Out-Null

function Invoke-PackageCommand([string]$Command) {
    $executable = Join-Path $isolatedPackage 'bin\jeh-hoho.exe'
    $output = & $executable $Command --home $isolatedPackage 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        throw "Packaged $Command command failed:`n$output"
    }
    return $output.Trim()
}

try {
    Get-ChildItem -LiteralPath $PackageDirectory -Force | Copy-Item -Destination $isolatedPackage -Recurse -Force
    $manifest = Get-Content -LiteralPath (Join-Path $isolatedPackage 'package-manifest.json') -Raw | ConvertFrom-Json
    foreach ($entry in $manifest.files) {
        $path = Join-Path $isolatedPackage ([string]$entry.path).Replace('/', '\')
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Package is missing $($entry.path)."
        }
        $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $entry.sha256) {
            throw "Package checksum failed for $($entry.path)."
        }
    }

    $forbidden = Get-ChildItem -LiteralPath $isolatedPackage -File -Recurse | Where-Object {
        $_.Extension -in @('.json', '.js', '.mjs', '.cmd', '.md', '.yaml', '.yml', '.proto')
    } | Select-String -Pattern 'tramuthus|Fin_FeedSat_1|fin_feed|feedsat' -CaseSensitive:$false
    if ($forbidden) {
        throw "Package contains a forbidden source/reference dependency: $($forbidden[0].Path)"
    }

    $status = Invoke-PackageCommand 'status'
    if ($status -notmatch '^(NOT STARTED|STOPPED)(\s|$)') {
        throw "Unexpected initial packaged status: $status"
    }
    $support = Invoke-PackageCommand 'support'
    if ($support -notmatch 'Support report created:') {
        throw "Packaged support command did not create a report: $support"
    }

    if ($FullWorkflow) {
        $credentialsPath = Join-Path $isolatedPackage 'config\credentials.json'
        if (-not (Test-Path -LiteralPath $credentialsPath -PathType Leaf)) {
            throw 'Full workflow requires config\credentials.json in the candidate package.'
        }
        $executable = Join-Path $isolatedPackage 'bin\jeh-hoho.exe'
        $stdoutPath = Join-Path $isolationRoot 'start.stdout.log'
        $stderrPath = Join-Path $isolationRoot 'start.stderr.log'
        $process = Start-Process -FilePath $executable -ArgumentList @('start', '--home', $isolatedPackage) -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
        $deadline = [DateTime]::UtcNow.AddSeconds(45)
        do {
            if ($process.HasExited) {
                $failure = (Get-Content -LiteralPath $stderrPath -Raw -ErrorAction SilentlyContinue) + (Get-Content -LiteralPath $stdoutPath -Raw -ErrorAction SilentlyContinue)
                throw "Packaged START exited before READY:`n$failure"
            }
            $status = Invoke-PackageCommand 'status'
            if ($status -match '^READY(\s|$)') { break }
        } while ([DateTime]::UtcNow -lt $deadline)
        if ($status -notmatch '^READY(\s|$)') {
            throw "Packaged product did not reach READY: $status"
        }
        $stop = Invoke-PackageCommand 'stop'
        if ($stop -notmatch 'stop requested') {
            throw "Packaged STOP was not accepted: $stop"
        }
        if (-not $process.WaitForExit(30000)) {
            throw 'Packaged product did not stop within 30 seconds.'
        }
        if ($process.ExitCode -ne 0) {
            throw "Packaged product exited with code $($process.ExitCode)."
        }
        $status = Invoke-PackageCommand 'status'
        if ($status -notmatch '^(NOT STARTED|STOPPED)(\s|$)') {
            throw "Unexpected final packaged status: $status"
        }
    }

    Write-Host "PASS: isolated package integrity and operator controls"
    if ($FullWorkflow) { Write-Host 'PASS: packaged START, READY, STATUS, and STOP workflow' }
} finally {
    if ($process -and -not $process.HasExited) {
        try { Invoke-PackageCommand 'stop' | Out-Null } catch {}
        if (-not $process.WaitForExit(10000)) { $process.Kill() }
    }
    Remove-Item -LiteralPath $isolationRoot -Recurse -Force -ErrorAction SilentlyContinue
}