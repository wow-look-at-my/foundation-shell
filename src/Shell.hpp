#pragma once

#include "mh/coroutine/task.hpp"

// Shell class with RAII initialization
class Shell
{
public:
	// Constructor initializes resources
	Shell();

	// Destructor cleans up resources
	~Shell();

	// Run the shell asynchronously
	mh::task<int> runAsync();

private:
	// Signal handlers
	void setupSignalHandlers();
};
