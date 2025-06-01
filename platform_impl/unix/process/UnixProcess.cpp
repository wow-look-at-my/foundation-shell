#include "UnixProcess.hpp"
#include "Config.hpp"
#include <iostream>
#include <cstring>
#include <signal.h>
#include <print>
#include <unistd.h>
#include <fcntl.h>
#include <sys/wait.h>
#include <mh/concurrency/dispatcher.hpp>
#include "core/GlobalDispatcher.hpp"
#include "LastCppInclude.hpp"


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

// Custom awaitable for process completion using SIGCHLD
struct ProcessWaitTask {
	int pid_;
	int* exitCode_;
	bool* completed_;
	
	// Global SIGCHLD self-pipe (shared across all processes)
	static int sigchld_pipe_[2];
	static bool pipe_initialized_;
	
	static void initSigchldPipe() {
		if (!pipe_initialized_) {
			if (pipe(sigchld_pipe_) == 0) {
				// Set write end to non-blocking
				int flags = fcntl(sigchld_pipe_[1], F_GETFL);
				fcntl(sigchld_pipe_[1], F_SETFL, flags | O_NONBLOCK);
				
				// Install SIGCHLD handler
				struct sigaction sa;
				sa.sa_handler = [](int) {
					char byte = 1;
					write(sigchld_pipe_[1], &byte, 1); // Wake up waiters
				};
				sigemptyset(&sa.sa_mask);
				sa.sa_flags = SA_RESTART | SA_NOCLDSTOP;
				sigaction(SIGCHLD, &sa, nullptr);
				
				pipe_initialized_ = true;
			}
		}
	}
	
	ProcessWaitTask(int pid, int* exitCode, bool* completed) 
		: pid_(pid), exitCode_(exitCode), completed_(completed) {
		initSigchldPipe();
	}
	
	bool await_ready() {
		// Check if process already exited
		int status;
		pid_t result = waitpid(pid_, &status, WNOHANG);
		if (result > 0) {
			// Process completed
			*completed_ = true;
			if (WIFEXITED(status)) {
				*exitCode_ = WEXITSTATUS(status);
			} else if (WIFSIGNALED(status)) {
				*exitCode_ = -WTERMSIG(status);
			} else {
				*exitCode_ = -1;
			}
			return true; // Don't suspend, we're done
		} else if (result == -1) {
			// Error occurred
			*completed_ = true;
			*exitCode_ = -1;
			return true; // Don't suspend, we're done
		}
		return false; // Suspend and wait for SIGCHLD
	}
	
	auto await_suspend(std::coroutine_handle<> handle) {
		if (!pipe_initialized_) {
			return handle; // Resume immediately if pipe not initialized
		}
		
		// Use the FD monitoring directly to wait for SIGCHLD
		auto fdTask = core::getGlobalDispatcher().co_wait_fd_read(sigchld_pipe_[0]);
		
		// Chain the FD task with our process checking
		return fdTask.await_suspend([this, handle](auto) {
			// This will be called when the FD becomes readable
			// Drain the pipe
			char buffer[256];
			read(sigchld_pipe_[0], buffer, sizeof(buffer));
			
			// Check our specific process
			int status;
			pid_t result = waitpid(pid_, &status, WNOHANG);
			if (result > 0) {
				// Our process completed
				*completed_ = true;
				if (WIFEXITED(status)) {
					*exitCode_ = WEXITSTATUS(status);
				} else if (WIFSIGNALED(status)) {
					*exitCode_ = -WTERMSIG(status);
				} else {
					*exitCode_ = -1;
				}
				handle.resume();
			} else if (result == -1) {
				// Error occurred
				*completed_ = true;
				*exitCode_ = -1;
				handle.resume();
			} else {
				// Not our process, wait again
				auto nextFdTask = core::getGlobalDispatcher().co_wait_fd_read(sigchld_pipe_[0]);
				return nextFdTask.await_suspend(/* recursive call */);
			}
		});
	}
	
	int await_resume() {
		return *exitCode_;
	}
};

// Static member definitions
int ProcessWaitTask::sigchld_pipe_[2] = {-1, -1};
bool ProcessWaitTask::pipe_initialized_ = false;

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

	// Initialize SIGCHLD handling if needed
	ProcessWaitTask::initSigchldPipe();
	
	// First check if process already exited
	int status;
	pid_t result = waitpid(pid_, &status, WNOHANG);
	if (result > 0) {
		// Process already completed
		completed_ = true;
		if (WIFEXITED(status)) {
			exitCode_ = WEXITSTATUS(status);
		} else if (WIFSIGNALED(status)) {
			exitCode_ = -WTERMSIG(status);
		} else {
			exitCode_ = -1;
		}
		co_return exitCode_;
	} else if (result == -1) {
		// Error occurred
		completed_ = true;
		exitCode_ = -1;
		co_return exitCode_;
	}
	
	// Process is still running, wait for SIGCHLD
	while (!completed_) {
		// Wait for SIGCHLD signal via FD monitoring
		co_await core::getGlobalDispatcher().co_wait_fd_read(ProcessWaitTask::sigchld_pipe_[0]);
		
		// Drain the pipe
		char buffer[256];
		read(ProcessWaitTask::sigchld_pipe_[0], buffer, sizeof(buffer));
		
		// Check our specific process
		result = waitpid(pid_, &status, WNOHANG);
		if (result > 0) {
			// Our process completed
			completed_ = true;
			if (WIFEXITED(status)) {
				exitCode_ = WEXITSTATUS(status);
			} else if (WIFSIGNALED(status)) {
				exitCode_ = -WTERMSIG(status);
			} else {
				exitCode_ = -1;
			}
			break;
		} else if (result == -1) {
			// Error occurred
			completed_ = true;
			exitCode_ = -1;
			break;
		}
		// If result == 0, not our process, wait for next SIGCHLD
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
