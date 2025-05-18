#pragma once

#include <string>
#include <vector>
#include <memory>
#include "../io/ISource.hpp"
#include "../io/ISink.hpp"
#include "../Task.hpp"

// Forward declarations
class IProcess;
using ProcessPtr = std::shared_ptr<IProcess>;

/**
 * Options for process creation
 */
struct ProcessOptions
{
	std::string command;
	std::vector<std::string> args;
	Source inputSource = nullptr;
	Sink outputSink = nullptr;
	Sink errorSink = nullptr;
};

/**
 * Interface for platform-agnostic process creation and management
 */
class IProcess
{
public:
	virtual ~IProcess() = default;

	/**
	 * Create a platform-specific process implementation
	 * @param options The process creation options
	 * @return Shared pointer to IProcess interface
	 */
	static ProcessPtr create(const ProcessOptions &options);

	/**
	 * Start the process
	 * @return True if process started successfully
	 */
	virtual bool start() = 0;

	/**
	 * Wait for the process to complete
	 * @return Exit code of the process
	 */
	virtual Task<int> waitAsync() = 0;

	/**
	 * Check if the process is running
	 * @return True if process is still running
	 */
	virtual bool isRunning() const = 0;

	/**
	 * Get the process ID
	 * @return Platform-specific process ID
	 */
	virtual intptr_t getPid() const = 0;

	/**
	 * Terminate the process
	 */
	virtual void terminate() = 0;
};

/**
 * Helper function to create a new process
 * @param command The command to execute
 * @param args Command arguments
 * @param inputSource Input source for the process (stdin)
 * @param outputSink Output sink for the process (stdout)
 * @param errorSink Error sink for the process (stderr)
 * @return A new process instance
 */
inline ProcessPtr createProcess(
	const std::string &command,
	const std::vector<std::string> &args,
	Source inputSource = nullptr,
	Sink outputSink = nullptr,
	Sink errorSink = nullptr)
{
	ProcessOptions options{command, args, inputSource, outputSink, errorSink};
	return IProcess::create(options);
}
