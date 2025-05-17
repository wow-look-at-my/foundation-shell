#pragma once

#include <string>
#include <vector>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <memory>

#undef stdin
#undef stdout
#undef stderr

// Platform-agnostic abstract handle type
using NativeHandle = intptr_t;

// Interface for data sources (e.g., stdout, stderr)
class ISource
{
public:
	virtual ~ISource() = default;

	// Read data from source
	virtual size_t read(void *buffer, size_t size) = 0;

	// Check if there's data available to read
	virtual bool canRead() const = 0;

	// Close the source
	virtual void close() = 0;

	// Get native handle (file descriptor on Unix, HANDLE on Windows)
	virtual NativeHandle getNativeHandle() const = 0;
};

// Interface for data sinks (e.g., stdin)
class ISink
{
public:
	virtual ~ISink() = default;

	// Write data to sink
	virtual size_t write(const void *buffer, size_t size) = 0;

	// Check if sink can accept data
	virtual bool canWrite() const = 0;

	// Flush any buffered data
	virtual void flush() = 0;

	// Close the sink
	virtual void close() = 0;

	// Get native handle (file descriptor on Unix, HANDLE on Windows)
	virtual NativeHandle getNativeHandle() const = 0;
};

// File-based implementation of ISource
class FileSource : public ISource
{
public:
	FileSource(const std::string &filename);
	~FileSource() override;

	size_t read(void *buffer, size_t size) override;
	bool canRead() const override;
	void close() override;
	NativeHandle getNativeHandle() const override { return handle; }

private:
	NativeHandle handle;
	bool closed = false;
};

// File-based implementation of ISink
class FileSink : public ISink
{
public:
	FileSink(const std::string &filename, bool append = false);
	~FileSink() override;

	size_t write(const void *buffer, size_t size) override;
	bool canWrite() const override;
	void flush() override;
	void close() override;
	NativeHandle getNativeHandle() const override { return handle; }

private:
	NativeHandle handle;
	bool closed = false;
};

// Pipe implementations
class PipeSource : public ISource
{
public:
	PipeSource(NativeHandle handle);
	~PipeSource() override;

	size_t read(void *buffer, size_t size) override;
	bool canRead() const override;
	void close() override;
	NativeHandle getNativeHandle() const override { return handle; }

private:
	NativeHandle handle;
	bool closed = false;
};

class PipeSink : public ISink
{
public:
	PipeSink(NativeHandle handle);
	~PipeSink() override;

	size_t write(const void *buffer, size_t size) override;
	bool canWrite() const override;
	void flush() override;
	void close() override;
	NativeHandle getNativeHandle() const override { return handle; }

private:
	NativeHandle handle;
	bool closed = false;
};

// Function to create a pipe pair
std::pair<std::shared_ptr<ISink>, std::shared_ptr<ISource>> createPipe();

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

	// Execute the command with optional I/O redirection
	bool execute() const;

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

// Function to execute a command chain
int executeCommandChain(const CommandChain &commandChain);

// Legacy function to parse input into commands with redirections and pipes (for backward compatibility)
std::vector<Command> parseCommand(const std::vector<std::string> &tokens);

// Function to split a string into tokens respecting quotes and escapes
std::vector<std::string> bashSplitString(const std::string &input);
