#!/usr/bin/env dotnet run

using System;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;

class Program
{
    static int Main(string[] args)
    {
        Console.WriteLine("Checking for proper upper CamelCase (PascalCase) naming convention...");

        var rootDir = args.Length > 0 ? args[0] : Directory.GetCurrentDirectory();
        var srcDir = Path.Combine(rootDir, "src");

        if (!Directory.Exists(srcDir))
        {
            Console.WriteLine($"Source directory not found: {srcDir}");
            return 1;
        }

        // Get all C++ files in the source directory (recursively)
        var cppFiles = Directory.GetFiles(srcDir, "*", SearchOption.AllDirectories)
            .Where(f => f.EndsWith(".cpp") || f.EndsWith(".hpp"))
            .ToArray();

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
                var properName = char.ToUpper(filename[0]) + filename.Substring(1);
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