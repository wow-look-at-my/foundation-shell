#!/usr/bin/env dotnet run

using System;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;

class Program
{
    static string ConvertToCamelCase(string filename)
    {
        var parts = filename.Split('_');
        var result = "";
        foreach (var part in parts)
        {
            if (part.Length > 0)
            {
                result += char.ToUpper(part[0]) + part.Substring(1);
            }
        }
        return result;
    }

    static int Main(string[] args)
    {
        Console.WriteLine("Checking for proper upper CamelCase (PascalCase) naming convention...");

        var rootDir = args.Length > 0 ? args[0] : Directory.GetCurrentDirectory();
        var directories = new[] { "src", "tests" };

        // Get all C++ files in the specified directories (recursively)
        var cppFiles = directories
            .Select(dir => Path.Combine(rootDir, dir))
            .Where(Directory.Exists)
            .SelectMany(dir => Directory.GetFiles(dir, "*", SearchOption.AllDirectories))
            .Where(f => f.EndsWith(".cpp") || f.EndsWith(".hpp"))
            .ToArray();

        Console.WriteLine($"Checking {cppFiles.Length} C++ files for proper upper CamelCase (PascalCase) naming...");

        // Counter for issues found
        int issuesCount = 0;

        foreach (var file in cppFiles)
        {
            // Get just the filename
            var filename = Path.GetFileName(file);

            // Skip interface files (starting with 'I' followed by an uppercase letter)
            if (Regex.IsMatch(filename, @"^I[A-Z]"))
            {
                continue;
            }

            // Check if the file starts with a lowercase letter (not PascalCase)
            if (Regex.IsMatch(filename, @"^[a-z]"))
            {
                Console.WriteLine($"Warning: Non-compliant file should start with uppercase letter: {file}");
                issuesCount++;

                // Create the properly cased filename for reference
                var properName = ConvertToCamelCase(filename);
                Console.WriteLine($"  Should be: {Path.GetDirectoryName(file)}/{properName}");
            }
        }

        if (issuesCount == 0)
        {
            Console.WriteLine("All files comply with upper CamelCase (PascalCase) naming convention.");
            return 0;
        }
        else
        {
            Console.WriteLine($"Found {issuesCount} files that don't follow upper CamelCase (PascalCase) naming convention.");
            Console.WriteLine("These files must be renamed to follow proper upper CamelCase (PascalCase) naming convention.");
            Console.WriteLine("Build will not proceed until file names are corrected.");
            return 1;
        }
    }
}