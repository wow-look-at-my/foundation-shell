using System;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;

class Program
{
	static int Main(string[] args)
	{
		var projectRoot = Path.GetFullPath(Path.Combine(Directory.GetCurrentDirectory(), ".."));
		var directories = new[] { "src", "platform_impl", "tests" };

		var cppFiles = directories
			.Select(dir => Path.Combine(projectRoot, dir))
			.Where(Directory.Exists)
			.SelectMany(dir => Directory.GetFiles(dir, "*.cpp", SearchOption.AllDirectories))
			.ToList();

		var errors = 0;
		var includePattern = new Regex(@"^\s*#include\s+");

		foreach (var file in cppFiles)
		{
			var relativePath = Path.GetRelativePath(projectRoot, file);

			try
			{
				var lines = File.ReadAllLines(file);
				var includeLines = lines
					.Select((line, index) => new { Line = line.Trim(), Index = index + 1 })
					.Where(x => includePattern.IsMatch(x.Line))
					.ToList();

				if (!includeLines.Any())
					continue; // No includes, skip

				var lastCppIncludeLine = includeLines
					.Where(x => x.Line.Contains("LastCppInclude.hpp"))
					.FirstOrDefault();

				if (lastCppIncludeLine == null)
				{
					Console.WriteLine($"  {relativePath}: Missing #include \"LastCppInclude.hpp\"");
					errors++;
					continue;
				}

				var lastIncludeLine = includeLines.Last();

				if (lastCppIncludeLine.Index != lastIncludeLine.Index)
				{
					Console.WriteLine($"  {relativePath}: LastCppInclude.hpp is not the last include (found at line {lastCppIncludeLine.Index}, but last include is at line {lastIncludeLine.Index})");
					errors++;
				}
			}
			catch (Exception ex)
			{
				Console.WriteLine($"  {relativePath}: Error reading file - {ex.Message}");
				errors++;
			}
		}

		if (errors > 0)
		{
			Console.WriteLine("LastCppInclude.hpp check failed:");
			Console.WriteLine($"Found {errors} errors in {cppFiles.Count} cpp files.");
			return 1;
		}
		else
		{
			Console.WriteLine($"All {cppFiles.Count} cpp files have LastCppInclude.hpp as the last include.");
			return 0;
		}
	}
}
