#include "StdioFix.hpp" // Must be first to handle stdio identifiers
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
				   std::print("\n"); });

		// Basic SIGCHLD handler for background processes
		signal(SIGCHLD, SIG_IGN);

		signal(SIGTSTP, [](int)
			   {
 					// Handle Ctrl+Z (suspend process)
 					std::print("\n"); });
	}
}

// Destructor with cleanup
Shell::~Shell()
{
	// Nothing to clean up at the moment
}

// Implements the main shell loop as a coroutine
Task<int> Shell::runAsync()
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

		// Split the input into tokens
		std::vector<std::string> tokens;
		std::istringstream iss(input);
		std::string token;
		while (iss >> token)
		{
			tokens.push_back(token);
		}

		// Parse the command with potential redirections, pipes, and command chains
		CommandChain commandChain(tokens);

		// Execute the command chain asynchronously
		lastExitStatus = co_await commandChain.executeAsync();
	}

	co_return lastExitStatus;
}
