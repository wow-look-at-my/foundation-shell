#pragma once

#include <fcntl.h>
#include <sys/wait.h>
#include <unistd.h>

#include <cstdio>
#include <cstring>
#include <string>

// Structure to hold separate stdout and stderr output
struct ShellOutput
{
	std::string stdout_output;
	std::string stderr_output;
};

// Helper function to execute a command in the shell and get output
std::string runShellCommand(const std::string& command, bool use_bash = false);

// Helper function to execute a command and get separate stdout/stderr
ShellOutput runShellCommandSeparate(const std::string& command, bool use_bash = false);

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string& shellOutput, const std::string& command);
