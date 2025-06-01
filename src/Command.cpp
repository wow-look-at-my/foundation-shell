#include "Command.hpp"
#include <filesystem>
#include <iostream>
#include <cstdio>
#include <cstring>
#include <wordexp.h>
#include <filesystem>
#include <signal.h>
#include <regex>
#include <dirent.h>
#include <sstream>
#include <algorithm>
#include <thread>
#include <chrono>
#include <mh/concurrency/dispatcher.hpp>
#include "io/FileSource.hpp"
#include "io/FileSink.hpp"
#include "Main.hpp"

// Command implementation
Command::Command()
	: appendOutput(false), appendError(false)
{
}

int Command::handleBuiltins() const
{
	if (args.empty())
	{
		return 0;
	}

	// Handle CD command
	if (args.at(0) == "cd")
	{
		try
		{
			std::filesystem::current_path(args.at(1));
			return 0; // Success
		}
		catch (const std::exception &e)
		{
			std::print(stderr, "Failed to change directory to {}: {}\n", args.at(1), e.what());
			return 1;
		}
	}

	// Handle exit command
	if (args.at(0) == "exit")
	{
		// Exit the shell
		exit(0);
	}

	// Add pwd as a built-in command
	if (args.at(0) == "pwd")
	{
		const std::filesystem::path cwd = std::filesystem::current_path();
		std::print("{}\n", cwd.c_str());
		return 0;
	}

	// Add clear as a built-in command
	if (args.at(0) == "clear")
	{
		// ANSI escape sequence to clear the screen
		std::cout << "\033[2J\033[H";
		return 0;
	}

	// This shell is stateless, so we don't implement history or aliases

	// This shell is stateless, so we don't implement job control

	// Add help command
	if (args.at(0) == "help")
	{
		std::string_view headerColor = "\033[1;34m"; // Bold blue
		std::string_view cmdColor = "\033[1;32m";	 // Bold green

		std::cout << headerColor << "Foundation Shell - Available Commands:" << Colors::COLOR_RESET << std::endl;
		std::cout << cmdColor << "  cd [dir]" << Colors::COLOR_RESET << " - Change directory" << std::endl;
		std::cout << cmdColor << "  pwd" << Colors::COLOR_RESET << " - Print working directory" << std::endl;
		std::cout << cmdColor << "  exit" << Colors::COLOR_RESET << " - Exit the shell" << std::endl;
		std::cout << cmdColor << "  clear" << Colors::COLOR_RESET << " - Clear the screen" << std::endl;
		std::cout << cmdColor << "  history [-c]" << Colors::COLOR_RESET << " - Show command history or clear it" << std::endl;
		std::cout << cmdColor << "  jobs" << Colors::COLOR_RESET << " - List active jobs" << std::endl;
		std::cout << cmdColor << "  fg [job_id]" << Colors::COLOR_RESET << " - Bring job to foreground" << std::endl;
		std::cout << cmdColor << "  bg [job_id]" << Colors::COLOR_RESET << " - Continue job in background" << std::endl;
		std::cout << cmdColor << "  alias [name=value]" << Colors::COLOR_RESET << " - Display or set command aliases" << std::endl;
		std::cout << cmdColor << "  unalias name" << Colors::COLOR_RESET << " - Remove an alias" << std::endl;
		std::cout << cmdColor << "  config [key] [value]" << Colors::COLOR_RESET << " - View or set configuration" << std::endl;
		std::cout << cmdColor << "  themes" << Colors::COLOR_RESET << " - List available themes" << std::endl;
		std::cout << cmdColor << "  debug [on|off|stats]" << Colors::COLOR_RESET << " - Toggle debug mode or show stats" << std::endl;
		std::cout << cmdColor << "  help" << Colors::COLOR_RESET << " - Display this help message" << std::endl;
		std::cout << std::endl;
		std::cout << headerColor << "Special Characters:" << Colors::COLOR_RESET << std::endl;
		std::cout << cmdColor << "  |" << Colors::COLOR_RESET << " - Pipe output of one command to another" << std::endl;
		std::cout << cmdColor << "  &&" << Colors::COLOR_RESET << " - Chain commands (execute next only if previous succeeds)" << std::endl;
		std::cout << cmdColor << "  > file" << Colors::COLOR_RESET << " - Redirect output to file" << std::endl;
		std::cout << cmdColor << "  >> file" << Colors::COLOR_RESET << " - Append output to file" << std::endl;
		std::cout << cmdColor << "  < file" << Colors::COLOR_RESET << " - Redirect input from file" << std::endl;
		std::cout << cmdColor << "  2> file" << Colors::COLOR_RESET << " - Redirect error output to file" << std::endl;
		std::cout << cmdColor << "  2>> file" << Colors::COLOR_RESET << " - Append error output to file" << std::endl;
		std::cout << cmdColor << "  &" << Colors::COLOR_RESET << " - Run command in background" << std::endl;
		std::cout << cmdColor << "  !n" << Colors::COLOR_RESET << " - Execute command from history" << std::endl;
		return 0;
	}

	// Not a builtin command
	return -1;
}

// These functions have been moved to platform-specific implementations
// in the process/ directory. The platform-agnostic IProcess interface
// now handles all process creation and management.

// Function to execute a command asynchronously
mh::task<bool> Command::executeAsync(Source inputSource, Sink outputSink) const
{
	if (args.empty())
	{
		co_return true; // Exit status 0 -> true
	}

	// Try to handle builtin commands first
	int builtinResult = handleBuiltins();
	if (builtinResult >= 0)
	{
		co_return builtinResult == 0; // Convert exit status to bool
	}

	// Create a process using the platform-agnostic interface
	Sink errorSink = nullptr; // Will use platform default if not provided

	// Handle I/O redirection using files
	if (!inputFile.empty())
	{
		// Create platform-agnostic FileSource
		inputSource = FileSource::create(inputFile);
		// For now, rely on process implementation to handle this
		// std::cerr << "Warning: File input redirection is platform-specific\n";
	}

	if (!outputFile.empty())
	{
		// Create platform-agnostic FileSink
		outputSink = FileSink::create(outputFile, appendOutput);
		// For now, rely on process implementation to handle this
		// std::cerr << "Warning: File output redirection is platform-specific\n";
	}

	if (!errorFile.empty())
	{
		// Create platform-agnostic FileSink
		outputSink = FileSink::create(outputFile, appendOutput);
		// For now, rely on process implementation to handle this
		// std::cerr << "Warning: File error redirection is platform-specific\n";
	}

	// Create the process with proper I/O redirection
	ProcessPtr process = createProcess(
		args.at(0),	 // Command
		args,		 // Arguments (including command)
		inputSource, // Input source
		outputSink,	 // Output sink
		errorSink	 // Error sink
	);

	// Start the process
	if (!process->start())
	{
		std::cerr << Colors::COLOR_RED << "Failed to start process: " << args.at(0) << Colors::COLOR_RESET << "\n";
		co_return false;
	}

	// Wait for the process to complete
	int exitCode = co_await process->waitAsync();
	co_return exitCode == 0; // Convert exit status to bool
}

// Helper function to expand tilde in a token
static std::string expandTilde(const std::string &token)
{
	if (!token.empty() && token[0] == '~')
	{
		const char *home = getenv("HOME");
		if (home)
		{
			if (token.length() == 1)
			{
				return home;
			}
			else if (token[1] == '/')
			{
				return std::string(home) + token.substr(1);
			}
		}
	}
	return token;
}

// Helper function to expand environment variables in a token
static std::string expandEnvironmentVariables(const std::string &token)
{
	std::string result = token;
	size_t pos = 0;

	// Handle variable expansion (skip escaped dollars marked with \x01)
	while ((pos = result.find('$', pos)) != std::string::npos)
	{
		// Check if this is an escaped dollar (marked with \x01)
		if (pos > 0 && result[pos - 1] == '\x01')
		{
			pos++; // Skip this escaped dollar
			continue;
		}

		if (pos + 1 < result.length())
		{
			size_t start = pos + 1;
			size_t end = start;

			// Find the end of the variable name
			while (end < result.length() &&
				   (std::isalnum(result[end]) || result[end] == '_'))
			{
				end++;
			}

			if (end > start)
			{
				std::string varName = result.substr(start, end - start);
				const char *varValue = getenv(varName.c_str());
				if (varValue)
				{
					result.replace(pos, end - pos, varValue);
					pos += std::strlen(varValue);
				}
				else
				{
					result.replace(pos, end - pos, "");
				}
			}
			else
			{
				pos++;
			}
		}
		else
		{
			pos++;
		}
	}

	// Clean up escape markers
	pos = 0;
	while ((pos = result.find("\x01$", pos)) != std::string::npos)
	{
		result.replace(pos, 2, "$"); // Replace \x01$ with literal $
		pos += 1;
	}

	return result;
}

// Structure to track token context during parsing
struct TokenContext
{
	std::string content;
	bool was_single_quoted = false;
};

// Helper function to tokenize input respecting quotes
static std::vector<TokenContext> tokenizeInput(const std::string &input)
{
	std::vector<TokenContext> tokens;
	TokenContext current_token;
	bool in_single_quotes = false;
	bool in_double_quotes = false;

	for (size_t i = 0; i < input.length(); ++i)
	{
		char c = input[i];

		if (c == '\\' && !in_single_quotes && i + 1 < input.length())
		{
			// Handle escape sequences
			char next_char = input[i + 1];
			if (next_char == '\\')
			{
				// \\ becomes \
				current_token.content += '\\';
				i++; // Skip the next character
			}
			else if (next_char == '$')
			{
				// \$ becomes literal $ (use special marker to prevent expansion)
				current_token.content += "\x01$"; // Use special marker
				i++;							  // Skip the next character
			}
			else if (next_char == ' ')
			{
				// Escaped space
				current_token.content += ' ';
				i++; // Skip the next character
			}
			else
			{
				// Other escapes - just add the escaped character
				current_token.content += next_char;
				i++; // Skip the next character
			}
		}
		else if (c == '\'' && !in_double_quotes)
		{
			if (!in_single_quotes)
			{
				current_token.was_single_quoted = true;
			}
			in_single_quotes = !in_single_quotes;
		}
		else if (c == '"' && !in_single_quotes)
		{
			in_double_quotes = !in_double_quotes;
		}
		else if (std::isspace(c) && !in_single_quotes && !in_double_quotes)
		{
			if (!current_token.content.empty())
			{
				tokens.push_back(current_token);
				current_token = TokenContext{};
			}
		}
		else
		{
			current_token.content += c;
		}
	}

	// Add the last token if any
	if (!current_token.content.empty())
	{
		tokens.push_back(current_token);
	}

	return tokens;
}

// Main function to split a string into tokens with bash-like expansion
std::vector<std::string> bashSplitString(const std::string &input)
{
	auto token_contexts = tokenizeInput(input);
	std::vector<std::string> result;

	for (const auto &ctx : token_contexts)
	{
		if (ctx.was_single_quoted)
		{
			// Single-quoted tokens: no expansions
			result.push_back(ctx.content);
		}
		else
		{
			// Double-quoted or unquoted tokens: perform expansions
			std::string expanded = expandTilde(ctx.content);
			expanded = expandEnvironmentVariables(expanded);
			result.push_back(expanded);
		}
	}

	return result;
}

// Helper function to identify token type
TokenType identifyToken(std::string_view token, bool isLastToken)
{
	if (token == "|")
		return TokenType::Pipe;
	if (token == "&&")
		return TokenType::And;
	if (token == "||")
		return TokenType::Or;
	if (token == "<")
		return TokenType::RedirectStdIn;
	if (token == ">")
		return TokenType::RedirectStdOut;
	if (token == ">>")
		return TokenType::RedirectStdOutAppend;
	if (token == "2>")
		return TokenType::RedirectStdErr;
	if (token == "2>>")
		return TokenType::RedirectStdErrAppend;
	return TokenType::Command;
}

// Levenshtein distance calculation between two strings
int levenshteinDistance(std::string_view s1, std::string_view s2)
{
	const std::size_t len1 = s1.size(), len2 = s2.size();
	std::vector<std::vector<int>> d(len1 + 1, std::vector<int>(len2 + 1));

	d.at(0).at(0) = 0;
	for (std::size_t i = 1; i <= len1; ++i)
		d[i].at(0) = i;
	for (std::size_t i = 1; i <= len2; ++i)
		d.at(0)[i] = i;

	for (std::size_t i = 1; i <= len1; ++i)
		for (std::size_t j = 1; j <= len2; ++j)
			d[i][j] = std::min({d[i - 1][j] + 1, d[i][j - 1] + 1, d[i - 1][j - 1] + (s1[i - 1] == s2[j - 1] ? 0 : 1)});

	return d[len1][len2];
}

// Function to find command suggestions
std::vector<std::string> findCommandSuggestions(const std::string &command)
{
	std::vector<std::string> suggestions;
	const std::vector<std::string_view> commonCommands = {
		"ls", "cd", "pwd", "echo", "cat", "grep", "find", "mkdir", "rm", "cp", "mv",
		"exit", "clear", "help", "man", "touch", "chmod", "chown", "sudo",
		"ps", "top", "kill", "config", "themes"};

	// Add built-in commands
	for (const auto &builtinCmd : commonCommands)
	{
		int distance = levenshteinDistance(command, builtinCmd);
		if (distance <= 3) // Fixed threshold value
		{
			suggestions.push_back(std::string{builtinCmd});
		}
	}

	// Check $PATH for executable commands
	const char *pathEnv = getenv("PATH");
	if (pathEnv)
	{
		std::string pathStr(pathEnv);
		std::istringstream pathStream(pathStr);
		std::string path;

		while (std::getline(pathStream, path, ':'))
		{
			try
			{
				// Fallback to manual directory reading if filesystem support is problematic
				DIR *dir = opendir(path.c_str());
				if (dir)
				{
					struct dirent *entry;
					while ((entry = readdir(dir)) != nullptr)
					{
						std::string_view filename = entry->d_name;
						// Skip . and ..
						if (filename == "." || filename == "..")
							continue;

						// Build full path for checking if it's executable
						std::string fullPath = path + "/" + std::string{filename};

						// Check if file exists and is executable
						auto status = std::filesystem::status(fullPath);
						if (std::filesystem::is_regular_file(fullPath) && (status.permissions() & std::filesystem::perms::owner_exec) != std::filesystem::perms::none)
						{
							int distance = levenshteinDistance(command, filename);
							if (distance <= 3)
							{
								suggestions.push_back(std::string{filename});
							}
						}
					}
					closedir(dir);
				}
			}
			catch (const std::exception &e)
			{
				// silently skip directories we can't read
			}
		}
	}

	// Remove duplicates
	std::sort(suggestions.begin(), suggestions.end());
	suggestions.erase(std::unique(suggestions.begin(), suggestions.end()), suggestions.end());

	return suggestions;
}
