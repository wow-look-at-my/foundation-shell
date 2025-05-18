#include "Shell.hpp"
#include <iostream>

int main()
{
	// Get the shell instance (initializes automatically through RAII)
	Shell shell;

	// Start the shell coroutine
	Task<int> shellTask = shell.runAsync();

	// Wait for the shell to complete
	shellTask.wait();

	// Return the shell's exit status
	return shellTask.get_result();
}
