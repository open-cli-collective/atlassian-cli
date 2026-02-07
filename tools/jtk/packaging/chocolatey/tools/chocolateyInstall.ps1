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
    throw "32-bit Windows is not supported. jtk requires 64-bit Windows."
}

$baseUrl = "https://github.com/open-cli-collective/atlassian-cli/releases/download/jtk-v${version}"
$zipFile = "jtk_${version}_windows_${arch}.zip"
$url = "${baseUrl}/${zipFile}"

Write-Host "Installing jtk ${version} for Windows ${arch}..."
Write-Host "URL: ${url}"
Write-Host "Checksum (SHA256): ${checksum}"

Install-ChocolateyZipPackage -PackageName $env:ChocolateyPackageName `
    -Url $url `
    -UnzipLocation $toolsDir `
    -Checksum $checksum `
    -ChecksumType 'sha256'

Write-Host "jira-ticket-cli installed successfully. Run 'jtk --help' to get started."
