#pragma once

#include <atomic>
#include <condition_variable>
#include <memory>
#include <mutex>
#include <thread>

#include "mh/coroutine/task.hpp"

// Forward declarations
namespace mh
{
class dispatcher;
}

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

	// Timeout thread for testing
	std::thread timeoutThread_;

	// Synchronization for interruptible timeout
	std::atomic<bool> shutdownRequested_{false};
	std::condition_variable timeoutCv_;
	std::mutex timeoutMutex_;

	// Dispatcher for async operations (registered for current thread)
	std::unique_ptr<mh::dispatcher> dispatcher_;
};
