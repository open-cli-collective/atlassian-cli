[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$Version,

    [Parameter(Mandatory)]
    [string]$ScriptPath,

    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9a-fA-F]{64}$')]
    [string]$ChecksumAmd64,

    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9a-fA-F]{64}$')]
    [string]$ChecksumArm64
)

$ErrorActionPreference = 'Stop'

$amd64Url = "https://github.com/open-cli-collective/atlassian-cli/releases/download/cfl-v$Version/cfl_${Version}_windows_amd64.zip"
$arm64Url = "https://github.com/open-cli-collective/atlassian-cli/releases/download/cfl-v$Version/cfl_${Version}_windows_arm64.zip"
$script = Get-Content $ScriptPath -Raw

foreach ($placeholder in 'URL_AMD64_PLACEHOLDER', 'URL_ARM64_PLACEHOLDER', 'CHECKSUM_AMD64_PLACEHOLDER', 'CHECKSUM_ARM64_PLACEHOLDER') {
    if (-not $script.Contains($placeholder)) { throw "Missing $placeholder in $ScriptPath" }
}

$script = $script.Replace('URL_AMD64_PLACEHOLDER', $amd64Url)
$script = $script.Replace('URL_ARM64_PLACEHOLDER', $arm64Url)
$script = $script.Replace('CHECKSUM_AMD64_PLACEHOLDER', $ChecksumAmd64)
$script = $script.Replace('CHECKSUM_ARM64_PLACEHOLDER', $ChecksumArm64)

if ($script -match 'URL_AMD64_PLACEHOLDER|URL_ARM64_PLACEHOLDER|CHECKSUM_AMD64_PLACEHOLDER|CHECKSUM_ARM64_PLACEHOLDER|ChocolateyPackageVersion|\$\{version\}') {
    throw "Rendered $ScriptPath still contains a URL/checksum placeholder or runtime package-version expression"
}
if (-not $script.Contains($amd64Url)) { throw "Rendered script is missing $amd64Url" }
if (-not $script.Contains($arm64Url)) { throw "Rendered script is missing $arm64Url" }
if (-not $script.Contains($ChecksumAmd64)) { throw "Rendered script is missing the AMD64 checksum" }
if (-not $script.Contains($ChecksumArm64)) { throw "Rendered script is missing the ARM64 checksum" }

Set-Content $ScriptPath $script
