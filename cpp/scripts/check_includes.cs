#!/usr/bin/env dotnet run

using System;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;

class Program
{
    static int Main(string[] args)
    {
        var rootDir = args.Length > 0 ? args[0] : Path.GetDirectoryName(Directory.GetCurrentDirectory());
        Console.WriteLine($"Checking include paths in {rootDir}...");

        // Find all C++ files
        var cppFiles = Directory.GetFiles(rootDir, "*", SearchOption.AllDirectories)
            .Where(f => (f.EndsWith(".cpp") || f.EndsWith(".hpp") || f.EndsWith(".h") || f.EndsWith(".cc")) &&
                       !f.Contains("\\build\\") && !f.Contains("/build/") &&
                       !f.Contains("\\extern\\") && !f.Contains("/extern/"))
            .ToArray();

        int errorCount = 0;

        foreach (var file in cppFiles)
        {
            try
            {
                var fileContent = File.ReadAllLines(file);
                var fileDir = Path.GetDirectoryName(file);

                for (int lineNumber = 0; lineNumber < fileContent.Length; lineNumber++)
                {
                    var line = fileContent[lineNumber];

                    // Check for include statements using regex
                    var match = Regex.Match(line, @"#include\s+""([^""]+)""");
                    if (match.Success)
                    {
                        var includePath = match.Groups[1].Value;

                        // Check for "../" in include paths
                        if (includePath.Contains("../"))
                        {
                            Console.WriteLine($"{file}:{lineNumber + 1}: Error: Include with '../' found: {includePath}");
                            errorCount++;
                        }

                        // Check if using a full path for a file in the same directory
                        if (includePath.Contains("/"))
                        {
                            var includedFile = includePath.Split('/').Last();
                            var localFilePath = Path.Combine(fileDir, includedFile);

                            if (File.Exists(localFilePath))
                            {
                                Console.WriteLine($"{file}:{lineNumber + 1}: Error: Include uses path for file in same directory: {includePath}");
                                errorCount++;
                            }
                        }
                    }
                }
            }
            catch (Exception)
            {
                Console.WriteLine($"Warning: Could not read {file} - might not be a text file");
                continue;
            }
        }

        if (errorCount > 0)
        {
            Console.WriteLine($"\nFound {errorCount} include path issues.");
            return 1;
        }
        else
        {
            Console.WriteLine("No include path issues found.");
            return 0;
        }
    }
}