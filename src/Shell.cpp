#include "Shell.hpp"

#include <pthread.h>
#include <unistd.h>

#include <chrono>
#include <csignal>
#include <cstdio>
#include <cstdlib>
#include <format>
#include <iostream>
#include <memory>
#include <mh/concurrency/dispatcher.hpp>
#include <sstream>
#include <string>

#include "Command.hpp"
#include "CommandChain.hpp"
#include "LastCppInclude.hpp"

// Constructor with RAII initialization
Shell::Shell() : dispatcher_(std::make_unique<mh::dispatcher>())
{
	// Register dispatcher for current thread
	dispatcher_->register_for_current_thread();
	// Setup signal handlers
	{
		signal(SIGINT, [](int) {
			// Just print a newline and return to the prompt
			std::print(stderr, "\n");
		});

		// Don't ignore SIGCHLD - we need it for waitpid() to work properly
		// signal(SIGCHLD, SIG_IGN);

		// No SIGTSTP handler needed - we don't support background processes
	}

	// Start timeout thread for testing
	timeoutThread_ = std::thread([this]() {
		// Name the thread for debugging
		pthread_setname_np("shell-timeout");
		
		// Wait for timeout or shutdown signal
		std::unique_lock<std::mutex> lock(timeoutMutex_);
		if (timeoutCv_.wait_for(lock, std::chrono::seconds(30), [this] { return shutdownRequested_.load(); }))
		{
			// Shutdown was requested before timeout
			return;
		}
		
		// Timeout reached without shutdown signal
		std::print(stderr, "\nShell timeout reached - exiting\n");
		exit(124); // Use exit code 124 for timeout
	});
}

// Destructor with cleanup
Shell::~Shell()
{
	// Signal timeout thread to shutdown and wait for it
	if (timeoutThread_.joinable())
	{
		{
			std::lock_guard<std::mutex> lock(timeoutMutex_);
			shutdownRequested_.store(true);
		}
		timeoutCv_.notify_one();
		timeoutThread_.join();
	}
}

// Implements the main shell loop as a coroutine
mh::task<int> Shell::runAsync()
{
	std::string input;
	int lastExitStatus = 0;

	// Check if running interactively (when stdin is a terminal)
	bool isInteractive = isatty(STDIN_FILENO);

	// Display welcome message only in interactive mode
	if (isInteractive)
	{
		std::print(stderr, "{}Welcome to Foundation Shell{}\n", Colors::COLOR_BOLD, Colors::COLOR_RESET);
	}

	while (true)
	{
		// Simple prompt with last exit status color - only in interactive mode
		if (isInteractive)
		{
			std::string prompt_color =
			    lastExitStatus == 0 ? std::string{Colors::COLOR_GREEN} : std::string{Colors::COLOR_RED};
			std::print(stderr, "{}${} ", prompt_color, Colors::COLOR_RESET);
		}

		// Get user input
		bool eofEncountered = false;
		// Use C-style input to avoid poisoned cin
		char* line = nullptr;
		size_t len = 0;
		ssize_t bytes_read = getline(&line, &len, stdin);
		if (bytes_read != -1)
		{
			input = std::string(line, bytes_read > 0 && line[bytes_read - 1] == '\n' ? bytes_read - 1 : bytes_read);
			free(line);
		}
		else
		{
			if (line)
				free(line);
		}

		if (bytes_read == -1)
		{
			// Handle EOF - check if we have partial input to process
			if (!input.empty())
			{
				// Process the partial line before exiting
				eofEncountered = true;
			}
			else
			{
				// Handle EOF (Ctrl+D) - only show message in interactive mode
				if (isInteractive)
				{
					std::print(stderr, "\nExiting shell\n");
				}
				break;
			}
		}
		else
		{
			// Skip empty input only if we got a complete line
			if (input.empty())
			{
				continue;
			}
		}

		try
		{
			// Split the input into tokens using bash-like expansion
			std::vector<std::string> tokens = bashSplitString(input);

			// Skip if no tokens were generated (e.g., only whitespace)
			if (tokens.empty())
			{
				if (eofEncountered)
				{
					break; // Exit if EOF and no commands to process
				}
				continue;
			}

			// Parse the command with potential redirections, pipes, and command chains
			CommandChain commandChain(tokens);

			// Execute the command chain asynchronously
			lastExitStatus = co_await commandChain.executeAsync();
		}
		catch (const std::exception& e)
		{
			// Any error should bail out and return to prompt with error status
			std::print(stderr, "{}Error: {}{}\n", Colors::COLOR_RED, e.what(), Colors::COLOR_RESET);
			lastExitStatus = 1;
		}

		// Exit after processing if we encountered EOF
		if (eofEncountered)
		{
			break;
		}
	}

	co_return lastExitStatus;
}
