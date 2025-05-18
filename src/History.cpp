#include "History.hpp"
#include "Config.hpp"
#include <fstream>
#include <iostream>
#include <cstdlib>
#include <algorithm>

// Function to get the history file path
std::string getHistoryFilePath()
{
	const char *homeDir = getenv("HOME");
	if (!homeDir)
	{
		return Constants::HISTORY_FILE_NAME; // Fallback to current directory
	}
	return std::string(homeDir) + "/" + Constants::HISTORY_FILE_NAME;
}

// Function to save a command to history file
void saveToHistory(const std::string &command)
{
	// Skip empty commands, whitespace, or exit commands for testing compatibility
	if (command.empty() ||
		command.find_first_not_of(" \t\n\r") == std::string::npos ||
		command == "exit")
	{
		return;
	}

	std::string historyPath = getHistoryFilePath();
	std::ofstream historyFile(historyPath, std::ios::app);

	if (historyFile.is_open())
	{
		historyFile << command << std::endl;
		historyFile.close();
	}
}

// Function to read history from file
std::vector<std::string> readHistory()
{
	std::vector<std::string> history;
	std::string historyPath = getHistoryFilePath();
	std::ifstream historyFile(historyPath);

	if (historyFile.is_open())
	{
		std::string line;
		while (std::getline(historyFile, line))
		{
			history.push_back(line);
		}
		historyFile.close();
	}

	return history;
}

// Function to clear history
void clearHistory()
{
	std::string historyPath = getHistoryFilePath();
	std::ofstream historyFile(historyPath, std::ios::trunc);
	historyFile.close();
}

// Function to find command by history number (!n notation)
std::string findCommandByHistoryNumber(int number)
{
	std::vector<std::string> history = readHistory();

	if (number <= 0 || number > static_cast<int>(history.size()))
	{
		return "";
	}

	return history[number - 1]; // Convert to 0-based index
}
