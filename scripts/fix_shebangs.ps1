#!/usr/bin/env pwsh

# Fix shebang lines in C# scripts that may have been corrupted by VS Code C# extension formatter

Write-Host 'Fixing shebang lines in C# checker scripts...'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# List of C# scripts that need shebang lines
$CsScripts = Get-ChildItem $ScriptDir -Filter '*.cs'

foreach ($script in $CsScripts)
{
	if (Test-Path $script)
	{
		# Read the first line
		$firstLine = Get-Content $script -First 1

		if ($firstLine -ne '#!/usr/bin/env dotnet run')
		{
			Write-Host "Fixing shebang in $(Split-Path -Leaf $script)"

			# Read all content
			$content = Get-Content $script

			# Check if the first line looks like a malformed shebang
			if ($firstLine -match '^#.*dotnet.*run' -or
				$firstLine -match '^#.*!/usr/bin/env' -or
				$firstLine -eq '# !/usr/bin/env dotnet run')
			{
				# Skip the malformed first line and replace it
				$newContent = @('#!/usr/bin/env dotnet run') + $content[1..($content.Length - 1)]
			}
			else
			{
				# Keep all content but prepend correct shebang with blank line
				$newContent = @('#!/usr/bin/env dotnet run', '') + $content
			}

			# Write the corrected content back
			$newContent | Set-Content $script -Encoding UTF8

			# Make executable (if on Unix-like system)
			if ($IsLinux -or $IsMacOS)
			{
				chmod +x $script
			}
		}
	}
 else
	{
		Write-Warning "Script not found: $script"
	}
}

Write-Host 'Shebang fix complete.'
