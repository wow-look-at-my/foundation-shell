#!/usr/bin/env dotnet run

using System;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;

class Program
{
	static int Main(string[] args)
	{
		Console.WriteLine("Checking for static_assert patterns and TODO comments...");

		var rootDir = args.Length > 0 ? args[0] : Directory.GetCurrentDirectory();
		var srcDir = Path.Combine(rootDir, "src");

		if (!Directory.Exists(srcDir))
		{
			Console.WriteLine($"Source directory not found: {srcDir}");
			return 1;
		}

		// Find all C++ files in the source directory
		var cppFiles = Directory.GetFiles(srcDir, "*", SearchOption.AllDirectories)
			.Where(f => f.EndsWith(".cpp") || f.EndsWith(".hpp"))
			.ToArray();

		// Counter for modified files
		int modifiedCount = 0;
		int todoCount = 0;

		foreach (var file in cppFiles)
		{
			bool fileModified = false;
			var content = File.ReadAllLines(file).ToList();

			// Check if the file contains the incorrect static_assert pattern
			if (content.Any(line => line.Contains("static_assert(true,")))
			{
				Console.WriteLine($"Found incorrect static_assert in {file}");

				// Mark file as modified
				fileModified = true;

				// Replace all occurrences of static_assert(true, with static_assert(false,
				for (int i = 0; i < content.Count; i++)
				{
					content[i] = content[i].Replace("static_assert(true,", "static_assert(false,");
				}

				// Write the content back to the file
				File.WriteAllLines(file, content);

				// Increment the modified count
				modifiedCount++;
			}

			// Check for TODOs in comments that should be static assertions
			// Look for // TODO: patterns that are not already part of static_assert
			if (content.Any(line => line.Contains("// TODO:")) &&
				!content.Any(line => line.Contains("// TODO:") && line.Contains("static_assert")))
			{
				Console.WriteLine($"Found TODO comments in {file}");

				// Mark file as modified
				fileModified = true;

				// Create a new list to hold the modified content
				var newContent = new List<string>();

				// Process the file line by line
				for (int i = 0; i < content.Count; i++)
				{
					var line = content[i];

					var todoMatch = Regex.Match(line, @"// TODO:(.*)");
					if (todoMatch.Success)
					{
						// Extract the TODO message
						var todoMessage = todoMatch.Groups[1].Value.Trim();

						// Get indentation of the original line
						var indentationMatch = Regex.Match(line, @"^(\s*)");
						var indentation = indentationMatch.Success ? indentationMatch.Groups[1].Value : "";

						// Don't write the original comment line, only add the static_assert
						// Add the static_assert line with proper indentation
						newContent.Add($"{indentation}static_assert(false, \"TODO:{todoMessage}\");");

						todoCount++;
					}
					else
					{
						// Pass through unchanged
						newContent.Add(line);
					}
				}

				// Replace the original file with the modified content
				File.WriteAllLines(file, newContent);

				// If this section modified the file and the previous section didn't
				if (!fileModified)
				{
					modifiedCount++;
				}
			}
		}

		if (modifiedCount == 0)
		{
			Console.WriteLine("No issues found.");
		}
		else
		{
			Console.WriteLine($"Fixed {modifiedCount} file(s):");
			Console.WriteLine("- Fixed incorrect static_assert patterns");
			Console.WriteLine($"- Converted {todoCount} TODO comments to static_assert statements");
		}

		return 0;
	}
}
