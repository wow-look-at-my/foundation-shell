#include "Shell.hpp"
#include "Command.hpp"
#include "CommandChain.hpp"
#include "History.hpp"
#include "Job.hpp"
#include "Alias.hpp"
#include <iostream>
#include <string>
#include <memory>

// External globals
extern bool debugMode;
extern DebugInfo debugInfo;
extern void sigchldHandler(int);

// Constructor with RAII initialization
Shell::Shell()
{
	// Initialize debug info
	debugInfo.startTime = std::chrono::steady_clock::now();

	// Setup signal handlers
	{
		signal(SIGINT, [](int)
			   {
				   // Just print a newline and return to the prompt
				   std::cout << std::endl; });

		signal(SIGCHLD, sigchldHandler);

		signal(SIGTSTP, [](int)
			   {
 					// Handle Ctrl+Z (suspend process)
 					std::cout << std::endl; });
	}
}

// Destructor with cleanup
Shell::~Shell()
{
	// Nothing to clean up at the moment
	// Future: save history, config changes, etc.
}

// Implements the main shell loop as a coroutine
Task<int> Shell::runAsync()
{
	std::string input;
	std::string prompt;
	int lastExitStatus = 0;

	// Display welcome message
	if (!shellConfig.welcomeMessage.empty())
	{
		std::cout << shellConfig.getThemeColor("header") << shellConfig.welcomeMessage
				  << Colors::COLOR_RESET << std::endl;
	}

	while (true)
	{
		// Format the prompt based on the template
		prompt = shellConfig.formatPrompt(shellConfig.promptTemplate, lastExitStatus);
		std::cout << prompt;

		// Get user input
		if (!std::getline(std::cin, input))
		{
			// Handle EOF (Ctrl+D)
			std::cout << "\nExiting shell\n";
			break;
		}

		// Skip empty input
		if (input.empty())
		{
			continue;
		}

		// Handle history execution with ! notation
		if (!input.empty() && input[0] == '!')
		{
			// Extract the history number
			int historyNum = 0;
			try
			{
				historyNum = std::stoi(input.substr(1));
				std::string historyCommand = findCommandByHistoryNumber(historyNum);
				if (!historyCommand.empty())
				{
					input = historyCommand;
					std::cout << input << std::endl; // Echo the command
				}
				else
				{
					std::cerr << "Event not found: " << historyNum << std::endl;
					continue;
				}
			}
			catch (const std::exception &e)
			{
				std::cerr << "Invalid history reference: " << input << std::endl;
				continue;
			}
		}

		// Handle aliases
		std::string expandedInput = expandAlias(input);
		if (expandedInput != input && debugMode)
		{
			std::cerr << "Debug: Expanded alias: " << input << " -> " << expandedInput << std::endl;
		}

		// Save the command to history
		saveToHistory(input); // Save original command, not expanded alias

		// Split the input into tokens, using bash-compatible splitting
		std::vector<std::string> tokens = bashSplitString(expandedInput);

		// Parse the command with potential redirections, pipes, and command chains
		CommandChain commandChain(tokens);

		// Execute the command chain asynchronously
		lastExitStatus = co_await commandChain.executeAsync(shellConfig);
	}

	co_return lastExitStatus;
}
