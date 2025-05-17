#include "../include/config.hpp"
#include "../include/command.hpp"
#include "../include/history.hpp"
#include "../include/job.hpp"
#include "../include/alias.hpp"
#include <iostream>
#include <string>
#include <signal.h>

// Global configuration object
ShellConfig shellConfig;

int main()
{
	std::string input;
	std::string prompt;
	int lastExitStatus = 0;

	// Initialize debug info
	debugInfo.startTime = std::chrono::steady_clock::now();

	// Load configuration and aliases
	shellConfig.loadConfig();
	loadAliases();

	// Setup signal handlers
	signal(SIGINT, [](int)
		   {
        // Just print a newline and return to the prompt
        std::cout << std::endl; });

	signal(SIGCHLD, sigchldHandler);

	signal(SIGTSTP, [](int)
		   {
        // Handle Ctrl+Z (suspend process)
        std::cout << std::endl; });

	// Display welcome message
	if (!shellConfig.welcomeMessage.empty())
	{
		std::cout << shellConfig.getThemeColor("header") << shellConfig.welcomeMessage << Colors::COLOR_RESET << std::endl;
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
		CommandChain commandChain = parseCommandChain(tokens);

		// Execute the command chain
		lastExitStatus = executeCommandChain(commandChain);
	}

	return 0;
}
