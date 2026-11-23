# hostnamefetch installer for Windows.
#
# Install, update, or uninstall the native hostnamefetch.exe.
#
# One-line install (default action is Install):
#   iwr -useb https://raw.githubusercontent.com/WispW/hostnamefetch/main/windows/install.ps1 | iex
#
# Update:
#   $s = (iwr -useb https://raw.githubusercontent.com/WispW/hostnamefetch/main/windows/install.ps1).Content
#   & ([scriptblock]::Create($s)) -Action Update
#
# Uninstall:
#   $s = (iwr -useb https://raw.githubusercontent.com/WispW/hostnamefetch/main/windows/install.ps1).Content
#   & ([scriptblock]::Create($s)) -Action Uninstall

[CmdletBinding()]
param(
    [ValidateSet('Install', 'Update', 'Uninstall')]
    [string]$Action = 'Install',

    [string]$InstallDir = '',

    [switch]$NoPath,

    [switch]$Force,

    # Used by CI and local tests. Leave empty to download from GitHub Releases.
    [string]$AssetPath = '',

    [string]$ChecksumPath = ''
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repo = 'WispW/hostnamefetch'
$AssetName = 'hostnamefetch.exe'
$ChecksumAssetName = 'hostnamefetch.exe.sha256'

if (-not $InstallDir) {
    if (-not $env:LOCALAPPDATA) {
        throw 'LOCALAPPDATA is not set; pass -InstallDir.'
    }
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\hostnamefetch'
}

$InstallDir = [IO.Path]::GetFullPath($InstallDir)
$TargetExe = Join-Path $InstallDir $AssetName
$PathSeparators = [char[]]@('\', '/')

if ($PSVersionTable.PSVersion.Major -lt 6) {
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    } catch {
        # The default protocol is enough on newer Windows builds.
    }
}

function Write-Step {
    param([string]$Message)
    Write-Host "hostnamefetch: $Message"
}

function Get-ExeVersion {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "not found: $Path"
    }

    $reported = & $Path --version
    if ($LASTEXITCODE -ne 0) {
        throw "version check failed for $Path"
    }

    $value = ($reported -join "`n").Trim()
    if ($value -match '^hostnamefetch\s+(.+)$') {
        return $Matches[1].Trim()
    }

    throw "unexpected --version output: $value"
}

function Convert-ToVersion {
    param([string]$Value)

    $clean = $Value.Trim() -replace '^v', ''
    try {
        return [version]$clean
    } catch {
        return $null
    }
}

function Test-NewerVersion {
    param(
        [string]$Current,
        [string]$Latest
    )

    $currentVersion = Convert-ToVersion $Current
    $latestVersion = Convert-ToVersion $Latest

    if ($null -eq $currentVersion -or $null -eq $latestVersion) {
        return $true
    }

    return $latestVersion -gt $currentVersion
}

function Read-Checksum {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "checksum file not found: $Path"
    }

    $text = Get-Content -LiteralPath $Path -Raw
    $match = [regex]::Match($text, '(?i)\b[0-9a-f]{64}\b')
    if (-not $match.Success) {
        throw "no SHA256 found in $Path"
    }

    return $match.Value.ToLowerInvariant()
}

function Get-UserPathEntries {
    $value = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ([string]::IsNullOrWhiteSpace($value)) {
        return @()
    }

    return @($value -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Test-PathEntry {
    param(
        [string[]]$Entries,
        [string]$Directory
    )

    $needle = $Directory.TrimEnd($PathSeparators)
    foreach ($entry in $Entries) {
        if ($entry.TrimEnd($PathSeparators) -ieq $needle) {
            return $true
        }
    }

    return $false
}

function Add-UserPathEntry {
    param([string]$Directory)

    $entries = Get-UserPathEntries
    if (-not (Test-PathEntry -Entries $entries -Directory $Directory)) {
        $newValue = (@($entries) + $Directory) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newValue, 'User')
    }

    if (-not $env:Path) {
        $env:Path = $Directory
        return
    }

    $processEntries = @($env:Path -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    if (-not (Test-PathEntry -Entries $processEntries -Directory $Directory)) {
        $env:Path = "$($env:Path);$Directory"
    }
}

function Remove-UserPathEntry {
    param([string]$Directory)

    $entries = Get-UserPathEntries
    if (Test-PathEntry -Entries $entries -Directory $Directory) {
        $needle = $Directory.TrimEnd($PathSeparators)
        $newValue = (@($entries | Where-Object { $_.TrimEnd($PathSeparators) -ine $needle }) -join ';')
        if ([string]::IsNullOrEmpty($newValue)) {
            [Environment]::SetEnvironmentVariable('Path', $null, 'User')
        } else {
            [Environment]::SetEnvironmentVariable('Path', $newValue, 'User')
        }
    }

    if (-not $env:Path) {
        return
    }

    $processEntries = @($env:Path -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $needle = $Directory.TrimEnd($PathSeparators)
    $kept = @($processEntries | Where-Object { $_.TrimEnd($PathSeparators) -ine $needle })
    if ($kept.Count -eq 0) {
        $env:Path = ''
    } else {
        $env:Path = $kept -join ';'
    }
}

function Get-ReleaseInfo {
    $headers = @{
        'User-Agent' = 'hostnamefetch-installer'
        'Accept' = 'application/vnd.github+json'
        'X-GitHub-Api-Version' = '2022-11-28'
    }

    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers $headers
    if ($release.draft -or $release.prerelease) {
        throw 'latest release is a draft or prerelease'
    }

    $asset = @($release.assets | Where-Object { $_.name -eq $AssetName } | Select-Object -First 1)
    if ($asset.Count -eq 0) {
        throw "release $($release.tag_name) has no $AssetName"
    }

    $checksumAsset = @($release.assets | Where-Object { $_.name -eq $ChecksumAssetName } | Select-Object -First 1)
    $checksum = $null
    if ($checksumAsset.Count -gt 0) {
        $checksum = $checksumAsset[0]
    }

    $digest = $null
    if ($null -ne $asset[0].PSObject.Properties['digest']) {
        $digest = [string]$asset[0].digest
    }

    return [pscustomobject]@{
        Tag = [string]$release.tag_name
        Asset = $asset[0]
        ChecksumAsset = $checksum
        Digest = $digest
    }
}

function Download-File {
    param(
        [string]$Uri,
        [string]$Path
    )

    Write-Step "downloading $Uri"
    Invoke-WebRequest -Uri $Uri -OutFile $Path -UseBasicParsing
}

function Get-ExpectedChecksum {
    param(
        $Release,
        [string]$ChecksumFile
    )

    if ($null -ne $Release) {
        if ($null -ne $Release.ChecksumAsset) {
            Download-File -Uri $Release.ChecksumAsset.browser_download_url -Path $ChecksumFile
            return Read-Checksum -Path $ChecksumFile
        }

        if ($Release.Digest -and $Release.Digest -match '(?i)^sha256:([0-9a-f]{64})$') {
            return $Matches[1].ToLowerInvariant()
        }

        throw 'release has no SHA256 (expected hostnamefetch.exe.sha256 or asset digest)'
    }

    if (-not $ChecksumFile) {
        throw 'ChecksumPath is required with AssetPath'
    }

    return Read-Checksum -Path $ChecksumFile
}

function Install-OrUpdate {
    if (-not (Test-Path -LiteralPath $InstallDir -PathType Container)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $installedVersion = $null
    if (Test-Path -LiteralPath $TargetExe -PathType Leaf) {
        try {
            $installedVersion = Get-ExeVersion -Path $TargetExe
        } catch {
            $installedVersion = $null
        }
    }

    $tempDir = Join-Path ([IO.Path]::GetTempPath()) ("hostnamefetch-" + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        $release = $null
        $sourceExe = $null
        $checksumFile = $null

        if ($AssetPath) {
            $sourceExe = [IO.Path]::GetFullPath($AssetPath)
            if (-not (Test-Path -LiteralPath $sourceExe -PathType Leaf)) {
                throw "asset not found: $sourceExe"
            }

            if ($ChecksumPath) {
                $checksumFile = [IO.Path]::GetFullPath($ChecksumPath)
            } else {
                $checksumFile = ''
            }

            $latestVersion = Get-ExeVersion -Path $sourceExe
        } else {
            $release = Get-ReleaseInfo
            $latestVersion = $release.Tag -replace '^v', ''
            $sourceExe = Join-Path $tempDir $AssetName
            Download-File -Uri $release.Asset.browser_download_url -Path $sourceExe
            $checksumFile = Join-Path $tempDir $ChecksumAssetName
        }

        if ($installedVersion -and -not $Force -and -not (Test-NewerVersion -Current $installedVersion -Latest $latestVersion)) {
            Write-Step "already up to date ($installedVersion)"
            return
        }

        $expected = Get-ExpectedChecksum -Release $release -ChecksumFile $checksumFile
        $actual = (Get-FileHash -LiteralPath $sourceExe -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            throw "SHA256 mismatch: expected $expected, got $actual"
        }

        $payloadVersion = Get-ExeVersion -Path $sourceExe
        if ($payloadVersion -ne $latestVersion) {
            throw "version mismatch: expected $latestVersion, got $payloadVersion"
        }

        try {
            Copy-Item -LiteralPath $sourceExe -Destination $TargetExe -Force
        } catch {
            throw "failed to install $TargetExe; close hostnamefetch.exe and retry. $($_.Exception.Message)"
        }

        Unblock-File -LiteralPath $TargetExe -ErrorAction SilentlyContinue

        if (-not $NoPath) {
            Add-UserPathEntry -Directory $InstallDir
        }

        Write-Step "installed $latestVersion to $InstallDir"
        if (-not $NoPath) {
            Write-Step 'open a new terminal to refresh PATH'
        }
    } finally {
        Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Uninstall {
    if (Test-Path -LiteralPath $InstallDir -PathType Container) {
        try {
            Remove-Item -LiteralPath $InstallDir -Recurse -Force
        } catch {
            throw "failed to remove $InstallDir; close hostnamefetch.exe and retry. $($_.Exception.Message)"
        }
    }

    Remove-UserPathEntry -Directory $InstallDir
    Write-Step "uninstalled from $InstallDir"
}

switch ($Action) {
    'Uninstall' {
        Uninstall
    }
    default {
        Install-OrUpdate
    }
}
