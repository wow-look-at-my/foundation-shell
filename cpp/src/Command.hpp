#pragma once

#include <chrono>
#include <memory>
#include <string>
#include <vector>

#include <mh/io/sink.hpp>
#include <mh/io/source.hpp>
#include <mh/process/process.hpp> // Include mh::process

#include "Config.hpp"            // Include ShellConfig
#include "TokenType.hpp"         // Include TokenType enum
#include "mh/coroutine/task.hpp" // Include the Task type

// Async sleep function using the dispatcher
mh::task<void> sleep_async(std::chrono::milliseconds duration);

// Helper function to split a string into tokens respecting quotes and escapes using wordexp
std::vector<std::string> bashSplitString(const std::string& input);

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

	// Execute the command with optional I/O redirection (async) - stateless version
	mh::task<bool> executeAsync(mh::io::source_ptr inputSource = nullptr, mh::io::sink_ptr outputSink = nullptr) const;

private:
	// Helper method to handle built-in commands - stateless version
	int handleBuiltins() const;
};
