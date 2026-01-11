#include <iostream>

#include <mh/concurrency/dispatcher.hpp>

#include "Shell.hpp"

#include "LastCppInclude.hpp"

// Implementation of sleep_async function using the thread dispatcher
mh::task<void> sleep_async(std::chrono::milliseconds duration)
{
	co_await mh::dispatcher::get().co_delay_for(duration);
}

#ifdef FOUNDATION_SHELL_EXE
int main()
{
	// Get the shell instance (initializes dispatcher automatically through RAII)
	Shell shell;

	// Start the shell coroutine
	mh::task<int> shellTask = shell.runAsync();

	// Start the dispatcher event loop
	while (!shellTask.is_ready())
	{
		// Process any pending events in the dispatcher
		mh::dispatcher::get().run_for(std::chrono::milliseconds(500));
	}

	// Return the shell's exit status
	return shellTask.get();
}
#endif
