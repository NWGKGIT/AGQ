[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$PackageIdentityName,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^CN=')]
    [string]$Publisher,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$PublisherDisplayName,

    [ValidatePattern('^\d+\.\d+\.\d+\.\d+$')]
    [string]$Version = '1.0.0.0',

    [string]$CertificatePath,
    [string]$CertificatePassword,
    [string]$TimestampUrl = 'http://timestamp.digicert.com',
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
$scriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path
$desktopDirectory = Resolve-Path (Join-Path $scriptDirectory '..\..')
$stagingDirectory = Join-Path $desktopDirectory 'build\msix\staging'
$outputDirectory = Join-Path $desktopDirectory 'build\msix'
$outputPackage = Join-Path $outputDirectory "AntigravityTokenMonitor-$Version-x64.msix"

function Resolve-WindowsSdkTool {
    param([Parameter(Mandatory = $true)][string]$Name)

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    $kitsRoot = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin'
    $candidate = Get-ChildItem -Path $kitsRoot -Filter $Name -Recurse -File -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match '\\x64\\' } |
        Sort-Object FullName -Descending |
        Select-Object -First 1
    if (-not $candidate) {
        throw "$Name was not found. Install the Windows 10/11 SDK."
    }
    return $candidate.FullName
}

function ConvertTo-XmlText {
    param([Parameter(Mandatory = $true)][string]$Value)
    return [System.Security.SecurityElement]::Escape($Value)
}

if (-not $SkipBuild) {
    Push-Location $desktopDirectory
    try {
        wails build -clean -trimpath -platform windows/amd64 -webview2 browser -o AntigravityTokenMonitor.exe
    }
    finally {
        Pop-Location
    }
}

$executable = Join-Path $desktopDirectory 'build\bin\AntigravityTokenMonitor.exe'
if (-not (Test-Path $executable -PathType Leaf)) {
    throw "Desktop executable is missing: $executable"
}

if (Test-Path $stagingDirectory) {
    Remove-Item -Path $stagingDirectory -Recurse -Force
}
New-Item -ItemType Directory -Path (Join-Path $stagingDirectory 'Assets') -Force | Out-Null
Copy-Item $executable (Join-Path $stagingDirectory 'AntigravityTokenMonitor.exe')
Copy-Item (Join-Path $scriptDirectory 'assets\*.png') (Join-Path $stagingDirectory 'Assets')

$manifest = Get-Content (Join-Path $scriptDirectory 'Package.appxmanifest.in') -Raw
$manifest = $manifest.Replace('@@PACKAGE_IDENTITY_NAME@@', (ConvertTo-XmlText $PackageIdentityName))
$manifest = $manifest.Replace('@@PUBLISHER@@', (ConvertTo-XmlText $Publisher))
$manifest = $manifest.Replace('@@PUBLISHER_DISPLAY_NAME@@', (ConvertTo-XmlText $PublisherDisplayName))
$manifest = $manifest.Replace('@@VERSION@@', $Version)
[System.IO.File]::WriteAllText(
    (Join-Path $stagingDirectory 'AppxManifest.xml'),
    $manifest,
    [System.Text.UTF8Encoding]::new($false)
)

$makeAppx = Resolve-WindowsSdkTool 'makeappx.exe'
if (Test-Path $outputPackage) {
    Remove-Item $outputPackage -Force
}
& $makeAppx pack /d $stagingDirectory /p $outputPackage /o
if ($LASTEXITCODE -ne 0) {
    throw "MakeAppx failed with exit code $LASTEXITCODE."
}

if ($CertificatePath) {
    if (-not (Test-Path $CertificatePath -PathType Leaf)) {
        throw "Signing certificate is missing: $CertificatePath"
    }
    $signTool = Resolve-WindowsSdkTool 'signtool.exe'
    $signArguments = @('sign', '/fd', 'SHA256', '/f', $CertificatePath)
    if ($CertificatePassword) {
        $signArguments += @('/p', $CertificatePassword)
    }
    if ($TimestampUrl) {
        $signArguments += @('/tr', $TimestampUrl, '/td', 'SHA256')
    }
    $signArguments += $outputPackage
    & $signTool @signArguments
    if ($LASTEXITCODE -ne 0) {
        throw "SignTool failed with exit code $LASTEXITCODE."
    }
}

$hash = Get-FileHash -Algorithm SHA256 $outputPackage
"$($hash.Hash.ToLowerInvariant())  $([System.IO.Path]::GetFileName($outputPackage))" |
    Set-Content -Encoding ascii "$outputPackage.sha256"

Write-Host "Created $outputPackage"
if (-not $CertificatePath) {
    Write-Warning 'The package is unsigned. Supply -CertificatePath for sideload testing; Partner Center release identity must match the certificate publisher.'
}
