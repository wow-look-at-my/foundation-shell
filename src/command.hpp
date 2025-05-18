#pragma once

#include <string>
#include <vector>
#include <memory>
#include "task.hpp"          // Include the Task type
#include "config.hpp"        // Include ShellConfig
#include "io/ISink.hpp"      // Include ISink interface
#include "io/ISource.hpp"    // Include ISource interface
#include "process/IProcess.hpp" // Include IProcess interface

// Token types for lexical analysis
enum class TokenType
{
	None,                       // No operator
	Command,                    // Command
	CommandArgument,
	Pipe,                       // | (pipe)
	And,                        // && (and)
	Or,                         // || (or)
	Background [[deprecated]],  // & (background)
	RedirectStdIn,              // < (input redirection)
	RedirectStdOut,             // > (output redirection)
	RedirectStdOutAppend,       // >> (append redirection)
	RedirectStdErr,             // 2> (error redirection)
	RedirectStdErrAppend,       // 2>> (error append redirection)
};

// Structure to represent a command with its I/O redirections
class Command
{
public:
	Command();
	~Command() = default;

	// Command arguments
	std::vector<std::string> args;

	// I/O redirection properties
	std::string inputFile;
	std::string outputFile;
	std::string errorFile;
	bool appendOutput = false;
	bool appendError = false;
	bool backgroundProcess = false;

	// I/O interfaces
	Sink stdin;
	Source stdout;
	Source stderr;

	// Execute the command with optional I/O redirection (async)
	Task<bool> executeAsync(Source inputSource = nullptr, Sink outputSink = nullptr) const;

private:
	// Helper method to handle built-in commands
	int handleBuiltins() const;
};

// Function to split a string into tokens respecting quotes and escapes
std::vector<std::string> bashSplitString(const std::string &input);