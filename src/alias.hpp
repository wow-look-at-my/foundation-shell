#pragma once

#include <string>
#include <map>

// Map to store aliases
extern std::map<std::string, std::string> aliases;

// Function to get the aliases file path
std::string getAliasesFilePath();

// Function to load aliases from file
void loadAliases();

// Function to save aliases to file
void saveAliases();

// Function to add an alias
void addAlias(const std::string& name, const std::string& command);

// Function to remove an alias
void removeAlias(const std::string& name);

// Function to expand aliases in a command
std::string expandAlias(const std::string& command);
