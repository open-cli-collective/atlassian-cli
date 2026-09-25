[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^[0-9]+\.[0-9]+\.[0-9]+$')]
    [string]$Version,

    [Parameter(Mandatory)]
    [ValidateScript({ Test-Path -LiteralPath $_ -PathType Leaf })]
    [string]$ChecksumsPath
)

$ErrorActionPreference = 'Stop'

$assetNames = [ordered]@{
    amd64 = "jtk_${Version}_windows_amd64.zip"
    arm64 = "jtk_${Version}_windows_arm64.zip"
}
$checksums = Get-Content -LiteralPath $ChecksumsPath
$parsed = [ordered]@{}

foreach ($architecture in $assetNames.Keys) {
    $assetName = $assetNames[$architecture]
    $pattern = "^(?<hash>[0-9a-fA-F]{64})\s+\*?$([regex]::Escape($assetName))\s*$"
    $assetMatches = @(
        foreach ($line in $checksums) {
            $match = [regex]::Match($line.Trim(), $pattern)
            if ($match.Success) { $match }
        }
    )
    if ($assetMatches.Count -ne 1) {
        throw "Expected exactly one SHA256 entry for $assetName, found $($assetMatches.Count)"
    }
    $parsed[$architecture] = $assetMatches[0].Groups['hash'].Value.ToLowerInvariant()
}

$parsed | ConvertTo-Json -Compress
