#pragma once

#include <string>
#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <cstring>
#include <cstdio>

// Helper function to execute a command in the shell and get output
std::string runShellCommand(const std::string &command);

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string &shellOutput, const std::string &command);
