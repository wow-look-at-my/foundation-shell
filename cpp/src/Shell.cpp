#include "Shell.hpp"

#include <chrono>
#include <csignal>
#include <cstdio>
#include <cstdlib>
#include <format>
#include <iostream>
#include <memory>
#include <sstream>
#include <string>

#include <unistd.h>

#include <mh/concurrency/dispatcher.hpp>

#include "Command.hpp"
#include "CommandChain.hpp"

#include "LastCppInclude.hpp"


namespace
{
struct Init
{
	Init()
	{
		// Initialize the dispatcher
		m_Dispatcher.register_for_current_thread();
		m_Timeout = timeout_async();
	}

private:
	mh::task<> timeout_async()
	{
		std::print(stderr, "Shell timeout started - waiting for 30 seconds\n");
		// Wait for 30 seconds before exiting the shell
		// This is a simple timeout mechanism and should be
		co_await m_Dispatcher.co_delay_for(std::chrono::seconds(30));
		std::print(stderr, "\nShell timeout reached - exiting\n");
		exit(124); // Use exit code 124 for timeout
	}

	mh::dispatcher m_Dispatcher{true};
	mh::task<> m_Timeout;

} const s_init;
} // namespace

// Constructor with RAII initialization
Shell::Shell()
{
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
}

// Destructor with cleanup
Shell::~Shell()
{
	// No cleanup needed for task-based timeout
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
