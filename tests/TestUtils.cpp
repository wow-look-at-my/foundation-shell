#include "TestUtils.hpp"

#include <fcntl.h>
#include <sys/wait.h>
#include <unistd.h>

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

#include "LastCppInclude.hpp"

// Helper function to execute a command in the shell and get output
std::string runShellCommand(const std::string& command, bool use_bash)
{
	use_bash = false;
	// Create pipes
	int stdin_pipe[2];
	int stdout_pipe[2];
	int stderr_pipe[2];

	if (pipe(stdin_pipe) == -1 || pipe(stdout_pipe) == -1 || pipe(stderr_pipe) == -1)
	{
		return "Error creating pipes";
	}

	// Fork a child process
	pid_t pid = fork();

	if (pid == -1)
	{
		return "Error forking process";
	}
	else if (pid == 0)
	{
		// Child process

		// Redirect stdin to read from stdin_pipe
		dup2(stdin_pipe[0], STDIN_FILENO);
		close(stdin_pipe[0]);
		close(stdin_pipe[1]);

		// Redirect stdout to write to stdout_pipe
		dup2(stdout_pipe[1], STDOUT_FILENO);
		close(stdout_pipe[0]);
		close(stdout_pipe[1]);

		// Redirect stderr to write to stderr_pipe
		dup2(stderr_pipe[1], STDERR_FILENO);
		close(stderr_pipe[0]);
		close(stderr_pipe[1]);

		// Execute the shell or bash
		if (use_bash)
		{
			execl("/bin/bash", "bash", nullptr);
		}
		else
		{
			execl("./foundation_shell", "foundation_shell", nullptr);
		}

		// If execl returns, there was an error
		perror("Error executing shell");
		exit(1);
	}

	// Parent process

	// Close unused pipe ends
	close(stdin_pipe[0]);
	close(stdout_pipe[1]);
	close(stderr_pipe[1]);

	// Write command to the shell's stdin
	std::string full_command = command + "\nexit\n";
	write(stdin_pipe[1], full_command.c_str(), full_command.size());
	close(stdin_pipe[1]);

	// Read output from the shell's stdout
	char buffer[4096];
	std::string result;
	ssize_t bytes;

	while ((bytes = read(stdout_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
	{
		buffer[bytes] = '\0';
		result += buffer;
	}
	close(stdout_pipe[0]);

	// Read output from the shell's stderr and append to result
	while ((bytes = read(stderr_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
	{
		buffer[bytes] = '\0';
		result += buffer;
	}
	close(stderr_pipe[0]);

	// Wait for the child to finish
	int status;
	waitpid(pid, &status, 0);

	return result;
}

// Helper function to execute a command and get separate stdout/stderr
ShellOutput runShellCommandSeparate(const std::string& command, bool use_bash)
{
	// Create pipes
	int stdin_pipe[2];
	int stdout_pipe[2];
	int stderr_pipe[2];

	if (pipe(stdin_pipe) == -1 || pipe(stdout_pipe) == -1 || pipe(stderr_pipe) == -1)
	{
		return {"Error creating pipes", "Error creating pipes"};
	}

	// Fork a child process
	pid_t pid = fork();

	if (pid == -1)
	{
		return {"Error forking process", "Error forking process"};
	}
	else if (pid == 0)
	{
		// Child process

		// Redirect stdin to read from stdin_pipe
		dup2(stdin_pipe[0], STDIN_FILENO);
		close(stdin_pipe[0]);
		close(stdin_pipe[1]);

		// Redirect stdout to write to stdout_pipe
		dup2(stdout_pipe[1], STDOUT_FILENO);
		close(stdout_pipe[0]);
		close(stdout_pipe[1]);

		// Redirect stderr to write to stderr_pipe
		dup2(stderr_pipe[1], STDERR_FILENO);
		close(stderr_pipe[0]);
		close(stderr_pipe[1]);

		// Execute the shell or bash
		if (use_bash)
		{
			execl("/bin/bash", "bash", nullptr);
		}
		else
		{
			execl("./foundation_shell", "foundation_shell", nullptr);
		}

		// If execl returns, there was an error
		perror("Error executing shell");
		exit(1);
	}

	// Parent process

	// Close unused pipe ends
	close(stdin_pipe[0]);
	close(stdout_pipe[1]);
	close(stderr_pipe[1]);

	// Write command to the shell's stdin
	std::string full_command = command + "\nexit\n";
	write(stdin_pipe[1], full_command.c_str(), full_command.size());
	close(stdin_pipe[1]);

	// Read output from the shell's stdout
	char buffer[4096];
	std::string stdout_result;
	std::string stderr_result;
	ssize_t bytes;

	while ((bytes = read(stdout_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
	{
		buffer[bytes] = '\0';
		stdout_result += buffer;
	}
	close(stdout_pipe[0]);

	// Read output from the shell's stderr separately
	while ((bytes = read(stderr_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
	{
		buffer[bytes] = '\0';
		stderr_result += buffer;
	}
	close(stderr_pipe[0]);

	// Wait for the child to finish
	int status;
	waitpid(pid, &status, 0);

	return {stdout_result, stderr_result};
}

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string& shellOutput, const std::string& command)
{
	std::istringstream stream(shellOutput);
	std::string line;
	bool foundCommand = false;
	std::string output;

	// Special case for history command since its output includes numbers
	if (command == "history")
	{
		bool inHistory = false;
		while (std::getline(stream, line))
		{
			// Look for the history command itself
			if (!inHistory && line == command)
			{
				inHistory = true;
				continue;
			}

			// If we're in the history section and find a line with a number followed by text,
			// it's likely a history entry
			if (inHistory && !line.empty() && ((std::isdigit(line[0]) && line == "  ") || line == "[32m"))
			{ // Also look for color codes
				output += line + "\n";
			}

			// Stop when we reach the exit command or a new prompt
			if (line == "exit" || (inHistory && line == " $ "))
			{
				break;
			}
		}
	}
	else
	{
		// Standard extraction for other commands
		while (std::getline(stream, line))
		{
			if (!foundCommand && line == command)
			{
				foundCommand = true;
				continue;
			}

			// Skip lines containing prompts (look for "$" which is part of all prompts)
			if (foundCommand && line != " $ " && line != "exit")
			{
				output += line + "\n";
			}

			if (line == "exit")
			{
				break;
			}
		}
	}

	// Remove trailing newline if present
	if (!output.empty() && output.back() == '\n')
	{
		output.pop_back();
	}

	return output;
}
