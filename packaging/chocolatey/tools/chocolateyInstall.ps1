$ErrorActionPreference = 'Stop'

$version = $env:ChocolateyPackageVersion
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# Checksums injected by release workflow - DO NOT EDIT MANUALLY
$checksumAmd64 = 'CHECKSUM_AMD64_PLACEHOLDER'
$checksumArm64 = 'CHECKSUM_ARM64_PLACEHOLDER'

# Architecture detection with ARM64 support
if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $arch = 'arm64'
    $checksum = $checksumArm64
} elseif ([Environment]::Is64BitOperatingSystem) {
    $arch = 'amd64'
    $checksum = $checksumAmd64
} else {
    throw "32-bit Windows is not supported. jira-ticket-cli requires 64-bit Windows."
}

$baseUrl = "https://github.com/open-cli-collective/jira-ticket-cli/releases/download/v${version}"
$zipFile = "jira-ticket-cli_${version}_windows_${arch}.zip"
$url = "${baseUrl}/${zipFile}"

Write-Host "Installing jira-ticket-cli ${version} for Windows ${arch}..."
Write-Host "URL: ${url}"
Write-Host "Checksum (SHA256): ${checksum}"

Install-ChocolateyZipPackage -PackageName $env:ChocolateyPackageName `
    -Url $url `
    -UnzipLocation $toolsDir `
    -Checksum $checksum `
    -ChecksumType 'sha256'

# Exclude non-executables from shimming
# Chocolatey auto-creates shims for .exe files; .ignore files prevent shimming other files
New-Item "$toolsDir\LICENSE.ignore" -Type File -Force | Out-Null
New-Item "$toolsDir\README.md.ignore" -Type File -Force | Out-Null

Write-Host "jira-ticket-cli installed successfully. Run 'jira-ticket-cli --help' to get started."
