#pragma once

#include "task.hpp"
#include "config.hpp"

// Shell class with RAII initialization
class Shell
{
public:
	// Constructor initializes resources
	Shell();

	// Destructor cleans up resources
	~Shell();

	// Run the shell asynchronously
	Task<int> runAsync();

private:
	// Signal handlers
	void setupSignalHandlers();

	// The shell's configuration
	ShellConfig shellConfig;
};
