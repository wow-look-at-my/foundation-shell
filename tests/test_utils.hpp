#pragma once

#include <string>
#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <cstring>
#include <cstdio>

// Structure to hold separate stdout and stderr output
struct ShellOutput
{
	std::string stdout_output;
	std::string stderr_output;
};

// Helper function to execute a command in the shell and get output
std::string runShellCommand(const std::string &command);

// Helper function to execute a command and get separate stdout/stderr
ShellOutput runShellCommandSeparate(const std::string &command);

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string &shellOutput, const std::string &command);
