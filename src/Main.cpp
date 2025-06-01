#include "Shell.hpp"
#include <iostream>
#include <mh/concurrency/dispatcher.hpp>

// Global dispatcher for async operations
mh::dispatcher g_dispatcher;

// Implementation of sleep_async function using the dispatcher
mh::task<void> sleep_async(std::chrono::milliseconds duration)
{
	co_await g_dispatcher.co_delay_for(duration);
}

#ifdef FOUNDATION_SHELL_EXE
int main()
{
	// Get the shell instance (initializes automatically through RAII)
	Shell shell;

	// Start the shell coroutine
	mh::task<int> shellTask = shell.runAsync();

	// Start the dispatcher event loop
	while (!shellTask.is_ready())
	{
		// Process any pending events in the dispatcher
		g_dispatcher.run_for(std::chrono::milliseconds(500));
	}

	// Return the shell's exit status
	return shellTask.get();
}
#endif
