#include "Shell.hpp"
#include "Command.hpp"
#include "CommandChain.hpp"
#include <iostream>
#include <cstdio>
#include <string>
#include <memory>
#include <format>
#include <cstdio>
#include <sstream>
#include <chrono>
#include <csignal>
#include <unistd.h>
#include "LastCppInclude.hpp"

// Constructor with RAII initialization
Shell::Shell()
{
	// Setup signal handlers
	{
		signal(SIGINT, [](int)
			   {
				   // Just print a newline and return to the prompt
				   std::print(stderr, "\n"); });

		// Don't ignore SIGCHLD - we need it for waitpid() to work properly
		// signal(SIGCHLD, SIG_IGN);

		// No SIGTSTP handler needed - we don't support background processes
	}

	// Start timeout thread for testing
	timeoutThread_ = std::thread([this]() {
		std::this_thread::sleep_for(std::chrono::seconds(30));
		std::print(stderr, "\nShell timeout reached - exiting\n");
		exit(124); // Use exit code 124 for timeout
	});
}

// Destructor with cleanup
Shell::~Shell()
{
	// Clean up timeout thread
	if (timeoutThread_.joinable())
	{
		timeoutThread_.detach(); // Let it finish or terminate naturally
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
			std::string prompt_color = lastExitStatus == 0 ? std::string{Colors::COLOR_GREEN} : std::string{Colors::COLOR_RED};
			std::print(stderr, "{}${} ", prompt_color, Colors::COLOR_RESET);
		}

		// Get user input
		bool eofEncountered = false;
		if (!std::getline(std::cin, input))
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
		catch (const std::exception &e)
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
