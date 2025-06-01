#!/usr/bin/env -S pwsh -NoProfile
# Script to:
# 1. Convert static_assert(true, ...) to static_assert(false, ...)
# 2. Convert // TODO: comments to static_assert(false, "TODO: ...") statements

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
Set-PSDebug -Strict
$PSNativeCommandUseErrorActionPreference = $true

Write-Host 'Checking for static_assert patterns and TODO comments...'

# Find all C++ files in the source directory
$cppFiles = Get-ChildItem -Path './src' -Recurse -Include '*.cpp', '*.hpp'

# Counter for modified files
$modifiedCount = 0
$todoCount = 0

foreach ($file in $cppFiles)
{
	$fileModified = $false
	$content = Get-Content -Path $file.FullName

	# Check if the file contains the incorrect static_assert pattern
	if ($content -match 'static_assert\(true,')
	{
		Write-Host "Found incorrect static_assert in $($file.FullName)"

		# Mark file as modified
		$fileModified = $true

		# Replace all occurrences of static_assert(true, with static_assert(false,
		$content = $content -replace 'static_assert\(true,', 'static_assert(false,'

		# Write the content back to the file
		Set-Content -Path $file.FullName -Value $content

		# Increment the modified count
		$modifiedCount++
	}

	# Check for TODOs in comments that should be static assertions
	# Look for // TODO: patterns that are not already part of static_assert
	if (($content -match '// TODO:') -and -not ($content -match '// TODO:.*static_assert'))
	{
		Write-Host "Found TODO comments in $($file.FullName)"

		# Mark file as modified
		$fileModified = $true

		# Create a new array to hold the modified content
		$newContent = @()

		# Process the file line by line
		for ($i = 0; $i -lt $content.Count; $i++)
		{
			$line = $content[$i]

			if ($line -match '// TODO:')
			{
				# Extract the TODO message
				$todoMatch = [regex]::Match($line, '// TODO:(.*)')
				$todoMessage = $todoMatch.Groups[1].Value.Trim()

				# Get indentation of the original line
				$indentation = ''
				if ($line -match '^(\s*)')
				{
					$indentation = $matches[1]
				}

				# Don't write the original comment line, only add the static_assert

				# Add the static_assert line with proper indentation
				$newContent += "$indentation" + "static_assert(false, `"TODO:$todoMessage`");"

				$todoCount++
			}
			else
			{
				# Pass through unchanged
				$newContent += $line
			}
		}

		# Replace the original file with the modified content
		Set-Content -Path $file.FullName -Value $newContent

		# If this section modified the file and the previous section didn't
		if (-not $fileModified)
		{
			$modifiedCount++
		}
	}
}

if ($modifiedCount -eq 0)
{
	Write-Host 'No issues found.'
}
else
{
	Write-Host "Fixed $modifiedCount file(s):"
	Write-Host '- Fixed incorrect static_assert patterns'
	Write-Host "- Converted $todoCount TODO comments to static_assert statements"
}

exit 0
