#pragma once

#include <cstdio>
#include <cstring>
#include <string>

#include <fcntl.h>
#include <sys/wait.h>
#include <unistd.h>

// Structure to hold separate stdout and stderr output
struct ShellOutput
{
	ShellOutput(std::string combined, std::string out, std::string err);

	std::string combined_output;
	std::string stdout_output;
	std::string stderr_output;

	// Conversion operator to allow implicit conversion to std::string
	operator std::string() const { return combined_output; }
};

// Helper function to execute a command in the shell and get output
ShellOutput runShellCommand(const std::string& command, bool use_bash = false);

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string& shellOutput, const std::string& command);
