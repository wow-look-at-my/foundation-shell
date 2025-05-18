#!/usr/bin/env -S pwsh -NoProfile
# Script to check for proper upper CamelCase (PascalCase) naming convention in source files

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
Set-PSDebug -Strict
$PSNativeCommandUseErrorActionPreference = $true

Write-Host 'Checking for proper upper CamelCase (PascalCase) naming convention...'

# Get all C++ files in the source directory (recursively)
$cppFiles = Get-ChildItem -Path ./src -Recurse -Include *.cpp, *.hpp

# Counter for issues found
$issuesCount = 0

foreach ($file in $cppFiles)
{
	# Get just the filename
	$filename = $file.Name

	# Skip interface files (starting with 'I' followed by an uppercase letter)
	if ($filename -cmatch '^I[A-Z]')
	{
		continue
	}

	# Check if the file starts with a lowercase letter (not PascalCase)
	if ($filename -cmatch '^[a-z]')
	{
		Write-Warning "Non-compliant file should start with uppercase letter: $($file.FullName) "
		$issuesCount++

		# Create the properly cased filename for reference
		$properName = $filename.Substring(0, 1).ToUpper() + $filename.Substring(1)
		Write-Output "  Should be: $($file.DirectoryName)/$properName"
	}
}

if ($issuesCount -eq 0)
{
	Write-Host 'All files comply with upper CamelCase (PascalCase) naming convention.'
	exit 0
}
else
{
	Write-Host "Found $issuesCount files that don't follow upper CamelCase (PascalCase) naming convention."
	Write-Host 'These files must be renamed to follow proper upper CamelCase (PascalCase) naming convention.'
	Write-Host 'Build will not proceed until file names are corrected.'
	exit 1  # Exit with error to indicate issues were found
}
