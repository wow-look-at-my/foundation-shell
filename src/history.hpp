#pragma once
static_assert(false, "File naming convention: This file should be renamed to History.hpp");

#include <string>
#include <vector>

// Get the history file path
std::string getHistoryFilePath();

// Save a command to history file
void saveToHistory(const std::string& command);

// Read history from file
std::vector<std::string> readHistory();

// Clear history
void clearHistory();

// Find command by history number (!n notation)
std::string findCommandByHistoryNumber(int number);
