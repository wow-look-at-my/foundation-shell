#!/usr/bin/env -S pwsh -NoProfile

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
Set-PSDebug -Strict
$PSNativeCommandUseErrorActionPreference = $true

# Set up terminal colors
$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$Reset = "`e[0m"

# Get the root directory (default to script parent directory)
$RootDir = $PSScriptRoot | Split-Path -Parent
Write-Host "Checking include paths in $RootDir..."

# Find all C++ files
$CppFiles = Get-ChildItem -Path $RootDir -Recurse -Include '*.cpp', '*.hpp', '*.h', '*.cc' |
	Where-Object { -not $_.FullName.Contains('\build\') -and -not $_.FullName.Contains('/build/') }

$ErrorCount = 0

foreach ($File in $CppFiles)
{
	$FileContent = Get-Content -Path $File.FullName -ErrorAction SilentlyContinue
	if ($null -eq $FileContent)
	{
		Write-Host "$($Yellow)Warning: Could not read $($File.FullName) - might not be a text file$($Reset)"
		continue
	}

	$FileDir = $File.DirectoryName

	$LineNumber = 0
	foreach ($Line in $FileContent)
	{
		$LineNumber++

		# Check for include statements using regex
		if ($Line -match '#include\s+"([^"]+)"')
		{
			$IncludePath = $Matches[1]

			# Check for "../" in include paths
			if ($IncludePath -match '\.\.\/')
			{
				Write-Host "$($File.FullName):${LineNumber}: $($Red)Error: Include with '../' found: $IncludePath$($Reset)"
				$ErrorCount++
			}

			# Check if using a full path for a file in the same directory
			if ($IncludePath -match '\/')
			{
				$IncludedFile = $IncludePath -split '/' | Select-Object -Last 1
				$LocalFilePath = Join-Path -Path $FileDir -ChildPath $IncludedFile

				if (Test-Path -Path $LocalFilePath)
				{
					Write-Host "$($File.FullName):${LineNumber}: $($Red)Error: Include uses path for file in same directory: $IncludePath$($Reset)"
					$ErrorCount++
				}
			}
		}
	}
}

if ($ErrorCount -gt 0)
{
	Write-Host "`n$($Red)Found $ErrorCount include path issues.$($Reset)"
	exit 1
}
else
{
	Write-Host "$($Green)No include path issues found.$($Reset)"
	exit 0
}
