#pragma once

#include <string>
#include <vector>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <memory>
#include "task.hpp"       // Include the Task type
#include "config.hpp"     // Include ShellConfig
#include "io/ISink.hpp"   // Include ISink interface
#include "io/ISource.hpp" // Include ISource interface

#undef stdin
#undef stdout
#undef stderr

// Structure to represent a command with its I/O redirections
class Command
{
public:
	Command();
	~Command() = default;

	// Command arguments
	std::vector<std::string> args;

	// I/O redirection properties
	std::string inputFile;
	std::string outputFile;
	std::string errorFile;
	bool appendOutput = false;
	bool appendError = false;
	bool backgroundProcess = false;

	// I/O interfaces
	std::shared_ptr<ISink> stdin;
	std::shared_ptr<ISource> stdout;
	std::shared_ptr<ISource> stderr;

	// Execute the command with optional I/O redirection (async)
	Task<bool> execute(int inputFd = STDIN_FILENO, int outputFd = STDOUT_FILENO) const;

private:
	// Helper method to handle built-in commands
	int handleBuiltins() const;

	// Common I/O setup for child process
	void setupChildIO(int inputFd, int outputFd) const;

	// Convert vector of strings to array of C-strings
	char **vectorToCharArray(const std::vector<std::string> &args) const;

	// Free memory allocated for char array
	void freeCharArray(char **array, int size) const;
};

// Token types for lexical analysis
enum class TokenType
{
	Command, // Command
	CommandArgument,
	Pipe,					   // | (pipe)
	And,					   // && (and)
	Or,						   // || (or)
	Background [[deprecated]], // & (background)
	RedirectStdIn,			   // < (input redirection)
	RedirectStdOut,			   // > (output redirection)
	RedirectStdOutAppend,	   // >> (append redirection)
	RedirectStdErr,			   // 2> (error redirection)
	RedirectStdErrAppend,	   // 2>> (error append redirection)
};

// Enum for command chain operators
enum class [[deprecated]] ChainOperator
{
	None, // No chaining operator
	Pipe, // | (pipe)
	And,  // && (and)
	Or	  // || (or)
};

// Structure to represent a command chain (pipeline, && chain, etc.)
struct CommandChain
{
	std::vector<Command> commands;
	std::vector<ChainOperator> operators; // operators[i] is the operator between commands[i] and commands[i+1]
};

// Function to parse input into commands with redirections and chains
CommandChain parseCommandChain(const std::vector<std::string> &tokens);

// Async version of executeCommandChain
Task<int> executeCommandChainAsync(CommandChain commandChain, const ShellConfig &config);

// Legacy function to parse input into commands with redirections and pipes (for backward compatibility)
std::vector<Command> parseCommand(const std::vector<std::string> &tokens);

// Function to split a string into tokens respecting quotes and escapes
std::vector<std::string> bashSplitString(const std::string &input);