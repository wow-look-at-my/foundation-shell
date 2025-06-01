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

// Constructor with RAII initialization
Shell::Shell()
{
	// Setup signal handlers
	{
		signal(SIGINT, [](int)
			   {
				   // Just print a newline and return to the prompt
				   std::print(stderr, "\n"); });

		// Basic SIGCHLD handler for background processes
		signal(SIGCHLD, SIG_IGN);

		// No SIGTSTP handler needed - we don't support background processes
	}
}

// Destructor with cleanup
Shell::~Shell()
{
	// Nothing to clean up at the moment
}

// Implements the main shell loop as a coroutine
mh::task<int> Shell::runAsync()
{
	std::string input;
	int lastExitStatus = 0;

	// Display welcome message
	std::print("{}Welcome to Foundation Shell{}\n", Colors::COLOR_BOLD, Colors::COLOR_RESET);

	while (true)
	{
		// Simple prompt with last exit status color
		std::string prompt_color = lastExitStatus == 0 ? std::string{Colors::COLOR_GREEN} : std::string{Colors::COLOR_RED};
		std::print("{}${} ", prompt_color, Colors::COLOR_RESET);

		// Get user input
		if (!std::getline(std::cin, input))
		{
			// Handle EOF (Ctrl+D)
			std::print("\nExiting shell\n");
			break;
		}

		// Skip empty input
		if (input.empty())
		{
			continue;
		}

		try
		{
			// Split the input into tokens using bash-like expansion
			std::vector<std::string> tokens = bashSplitString(input);

			// Parse the command with potential redirections, pipes, and command chains
			CommandChain commandChain(tokens);

			// Execute the command chain asynchronously
			lastExitStatus = co_await commandChain.executeAsync();
		}
		catch (const std::exception &e)
		{
			// Any error should bail out and return to prompt with error status
			std::print("{}Error: {}{}\n", Colors::COLOR_RED, e.what(), Colors::COLOR_RESET);
			lastExitStatus = 1;
		}
	}

	co_return lastExitStatus;
}
