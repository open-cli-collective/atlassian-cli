$ErrorActionPreference = 'Stop'

$version = $env:ChocolateyPackageVersion
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# Architecture detection with ARM64 support
if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    $arch = 'arm64'
} elseif ([Environment]::Is64BitOperatingSystem) {
    $arch = 'amd64'
} else {
    throw "32-bit Windows is not supported. confluence-cli requires 64-bit Windows."
}

$baseUrl = "https://github.com/open-cli-collective/confluence-cli/releases/download/v${version}"
$zipFile = "cfl_${version}_windows_${arch}.zip"
$url = "${baseUrl}/${zipFile}"
$checksumsUrl = "${baseUrl}/checksums.txt"

# Fetch checksums and extract the one for our architecture
$checksums = (Invoke-WebRequest -Uri $checksumsUrl -UseBasicParsing).Content
$checksumLine = $checksums -split "`n" | Where-Object { $_ -match $zipFile }
if (-not $checksumLine) {
    throw "Could not find checksum for ${zipFile} in checksums.txt"
}
$checksum = ($checksumLine -split '\s+')[0]

Write-Host "Installing confluence-cli ${version} for Windows ${arch}..."
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

Write-Host "confluence-cli installed successfully. Run 'cfl --help' to get started."
