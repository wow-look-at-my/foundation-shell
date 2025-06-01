#pragma once

#include <string>
#include <vector>
#include <memory>
#include <chrono>
#include "Task.hpp"				// Include the Task type
#include "Config.hpp"			// Include ShellConfig
#include "io/ISink.hpp"			// Include ISink interface
#include "io/ISource.hpp"		// Include ISource interface
#include "process/IProcess.hpp" // Include IProcess interface
#include "TokenType.hpp"		// Include TokenType enum

#undef stdin
#undef stdout
#undef stderr

// Async sleep function using the dispatcher
Task<void> sleep_async(std::chrono::milliseconds duration);

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

	// I/O interfaces
	Sink stdin;
	Source stdout;
	Source stderr;

	// Execute the command with optional I/O redirection (async) - stateless version
	Task<bool> executeAsync(Source inputSource = nullptr, Sink outputSink = nullptr) const;

private:
	// Helper method to handle built-in commands - stateless version
	int handleBuiltins() const;
};
