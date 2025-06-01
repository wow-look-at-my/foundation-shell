#include "UnixProcess.hpp"
#include "Config.hpp"
#include <iostream>
#include <cstring>
#include <signal.h>
#include <print>

// Poison iostream to prevent accidental use - use std::print instead
#define cout DO_NOT_USE_COUT_USE_STD_PRINT_INSTEAD
#define cerr DO_NOT_USE_CERR_USE_STD_PRINT_STDERR_INSTEAD

// Constructor
UnixProcess::UnixProcess(
	const std::string &command,
	const std::vector<std::string> &args,
	Source inputSource,
	Sink outputSink,
	Sink errorSink)
	: command_(command),
	  args_(args),
	  inputSource_(inputSource),
	  outputSink_(outputSink),
	  errorSink_(errorSink),
	  pid_(0),
	  started_(false),
	  completed_(false),
	  exitCode_(0)
{
	// Ensure command is first in args list
	if (args_.empty() || args_[0] != command_)
	{
		args_.insert(args_.begin(), command_);
	}
}

// Destructor
UnixProcess::~UnixProcess()
{
	// Terminate the process if it's still running
	if (started_ && !completed_)
	{
		terminate();
	}
}

// Start the process
bool UnixProcess::start()
{
	if (started_)
	{
		return false; // Already started
	}

	// Fork a new process
	pid_ = fork();

	if (pid_ == -1)
	{
		// Fork failed
		std::print(stderr, ErrorMessages::FAILED_TO_FORK);
		return false;
	}
	else if (pid_ == 0)
	{
		// Child process

		// Setup I/O redirection
		setupChildIO();

		// Convert args to C-style array
		char **argArray = vectorToCharArray(args_);

		// Execute the command
		execvp(command_.c_str(), argArray);

		// If execvp returns, an error occurred
		std::print(stderr, ErrorMessages::COMMAND_NOT_FOUND, command_);
		freeCharArray(argArray, args_.size());
		exit(1);
	}
	else
	{
		// Parent process
		started_ = true;
		return true;
	}
}

// Wait for the process to complete
mh::task<int> UnixProcess::waitAsync()
{
	if (!started_)
	{
		co_return -1; // Not started
	}

	if (completed_)
	{
		co_return exitCode_; // Already completed
	}

	// Wait for the child process
	int status;
	pid_t result = waitpid(pid_, &status, 0);

	if (result == -1)
	{
		// Error occurred (suppress error message for now to avoid test pollution)
		// std::print(stderr, ErrorMessages::ERROR_WAITING_FOR_PROCESS);
		completed_ = true;
		exitCode_ = -1;
	}
	else
	{
		// Process completed
		completed_ = true;
		if (WIFEXITED(status))
		{
			exitCode_ = WEXITSTATUS(status);
		}
		else if (WIFSIGNALED(status))
		{
			exitCode_ = -WTERMSIG(status);
		}
		else
		{
			exitCode_ = -1;
		}
	}

	co_return exitCode_;
}

// Check if the process is running
bool UnixProcess::isRunning() const
{
	if (!started_ || completed_)
	{
		return false;
	}

	// Check if process exists
	if (kill(pid_, 0) == 0)
	{
		return true;
	}

	// Process doesn't exist
	return false;
}

// Get the process ID
intptr_t UnixProcess::getPid() const
{
	return static_cast<intptr_t>(pid_);
}

// Terminate the process
void UnixProcess::terminate()
{
	if (!started_ || completed_)
	{
		return;
	}

	// Send SIGTERM signal
	kill(pid_, SIGTERM);

	// Wait for process to terminate
	int status;
	waitpid(pid_, &status, 0);

	completed_ = true;
	if (WIFEXITED(status))
	{
		exitCode_ = WEXITSTATUS(status);
	}
	else if (WIFSIGNALED(status))
	{
		exitCode_ = -WTERMSIG(status);
	}
	else
	{
		exitCode_ = -1;
	}
}

// Setup I/O redirection for child process
void UnixProcess::setupChildIO() const
{
	// Handle input (stdin)
	if (inputSource_)
	{
		// Get native handle from Source
		int inputFd = static_cast<int>(inputSource_->getNativeHandle());
		dup2(inputFd, STDIN_FILENO);
		if (inputFd != STDIN_FILENO)
		{
			close(inputFd);
		}
	}

	// Handle output (stdout)
	if (outputSink_)
	{
		// Get native handle from Sink
		int outputFd = static_cast<int>(outputSink_->getNativeHandle());
		dup2(outputFd, STDOUT_FILENO);
		if (outputFd != STDOUT_FILENO)
		{
			close(outputFd);
		}
	}

	// Handle error (stderr)
	if (errorSink_)
	{
		// Get native handle from Sink
		int errorFd = static_cast<int>(errorSink_->getNativeHandle());
		dup2(errorFd, STDERR_FILENO);
		if (errorFd != STDERR_FILENO)
		{
			close(errorFd);
		}
	}
}

// Convert vector of strings to array of C-strings
char **UnixProcess::vectorToCharArray(const std::vector<std::string> &args) const
{
	char **result = new char *[args.size() + 1]; // +1 for the NULL terminator

	for (size_t i = 0; i < args.size(); i++)
	{
		result[i] = new char[args[i].size() + 1];
		std::strcpy(result[i], args[i].c_str());
	}

	result[args.size()] = nullptr; // Null-terminate the array
	return result;
}

// Free memory allocated for char array
void UnixProcess::freeCharArray(char **array, int size) const
{
	for (int i = 0; i < size; i++)
	{
		delete[] array[i];
	}
	delete[] array;
}

// Factory function implementation moved to UnixProcessFactory.cpp
