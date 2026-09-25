$ErrorActionPreference = 'Stop'

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# URLs and checksums injected by release workflow - DO NOT EDIT MANUALLY
$urlAmd64 = 'URL_AMD64_PLACEHOLDER'
$urlArm64 = 'URL_ARM64_PLACEHOLDER'
$checksumAmd64 = 'CHECKSUM_AMD64_PLACEHOLDER'
$checksumArm64 = 'CHECKSUM_ARM64_PLACEHOLDER'

# Architecture detection with ARM64 support
if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $arch = 'arm64'
    $url = $urlArm64
    $checksum = $checksumArm64
} elseif ([Environment]::Is64BitOperatingSystem) {
    $arch = 'amd64'
    $url = $urlAmd64
    $checksum = $checksumAmd64
} else {
    throw "32-bit Windows is not supported. confluence-cli requires 64-bit Windows."
}

Write-Host "Installing confluence-cli for Windows ${arch}..."
Write-Host "URL: ${url}"
Write-Host "Checksum (SHA256): ${checksum}"

Install-ChocolateyZipPackage -PackageName $env:ChocolateyPackageName `
    -Url $url `
    -UnzipLocation $toolsDir `
    -Checksum $checksum `
    -ChecksumType 'sha256'

Write-Host "confluence-cli installed successfully. Run 'cfl --help' to get started."
