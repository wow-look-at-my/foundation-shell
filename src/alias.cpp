#include "alias.hpp"
#include "config.hpp"
#include <fstream>
#include <iostream>
#include <sstream>
#include <cstdlib>
static_assert(false, "File naming convention: This file should be renamed to Alias.cpp");

// Initialize the aliases map
std::map<std::string, std::string> aliases;

// External variable for debug mode
extern bool debugMode;

// Function to get the aliases file path
std::string getAliasesFilePath() {
    const char* homeDir = getenv("HOME");
    if (!homeDir) {
        return Constants::ALIASES_FILE_NAME; // Fallback to current directory
    }
    return std::string(homeDir) + "/" + Constants::ALIASES_FILE_NAME;
}

// Function to load aliases from file
void loadAliases() {
    std::string aliasesPath = getAliasesFilePath();
    std::ifstream aliasesFile(aliasesPath);

    aliases.clear(); // Clear existing aliases

    if (!aliasesFile.is_open()) {
        if (debugMode) std::cerr << "Debug: No aliases file found at " << aliasesPath << std::endl;
        return;
    }

    std::string line;
    while (std::getline(aliasesFile, line)) {
        // Skip empty lines and comments
        if (line.empty() || line[0] == '#') {
            continue;
        }

        // Parse alias definition
        size_t equalsPos = line.find('=');
        if (equalsPos != std::string::npos) {
            std::string name = line.substr(0, equalsPos);
            std::string command = line.substr(equalsPos + 1);

            // Remove whitespace
            name.erase(0, name.find_first_not_of(" \t"));
            name.erase(name.find_last_not_of(" \t") + 1);
            command.erase(0, command.find_first_not_of(" \t"));
            command.erase(command.find_last_not_of(" \t") + 1);

            // Remove quotes if present
            if (command.size() >= 2 &&
                ((command.front() == '"' && command.back() == '"') ||
                 (command.front() == '\'' && command.back() == '\''))) {
                command = command.substr(1, command.size() - 2);
            }

            aliases[name] = command;
            if (debugMode) std::cerr << "Debug: Loaded alias '" << name << "' = '" << command << "'" << std::endl;
        }
    }

    aliasesFile.close();
}

// Function to save aliases to file
void saveAliases() {
    std::string aliasesPath = getAliasesFilePath();
    std::ofstream aliasesFile(aliasesPath);

    if (aliasesFile.is_open()) {
        aliasesFile << "# Foundation Shell Aliases File\n\n";
        for (const auto& [name, command] : aliases) {
            aliasesFile << name << "=" << command << "\n";
        }
        aliasesFile.close();
    }
}

// Function to add an alias
void addAlias(const std::string& name, const std::string& command) {
    aliases[name] = command;
    saveAliases();
}

// Function to remove an alias
void removeAlias(const std::string& name) {
    aliases.erase(name);
    saveAliases();
}

// Function to expand aliases in a command
std::string expandAlias(const std::string& command) {
    std::istringstream iss(command);
    std::string firstWord;
    iss >> firstWord;

    auto it = aliases.find(firstWord);
    if (it != aliases.end()) {
        // Get the rest of the original command
        std::string remainder = command.substr(command.find(firstWord) + firstWord.length());
        // Replace the alias with its definition and append the remainder
        return it->second + remainder;
    }

    return command;
}
