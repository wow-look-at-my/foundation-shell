#pragma once

#include "IProcess.hpp"
#include <string>
#include <vector>
#include <memory>

/**
 * Factory class for creating Process objects
 * This abstract factory allows platform-specific implementations to be created
 * without direct dependencies on platform-specific code
 */
class ProcessFactory
{
public:
	/**
	 * Get the singleton instance of the factory
	 */
	static ProcessFactory &getInstance();

	/**
	 * Create a process using the specified options
	 * @param command The command to execute
	 * @param args Command arguments
	 * @param inputSource Input source for the process (stdin)
	 * @param outputSink Output sink for the process (stdout)
	 * @param errorSink Error sink for the process (stderr)
	 * @return Shared pointer to IProcess interface
	 */
	virtual std::shared_ptr<IProcess> createProcess(
		const std::string &command,
		const std::vector<std::string> &args,
		Source inputSource,
		Sink outputSink,
		Sink errorSink) = 0;

protected:
	// Protected constructor for singleton pattern
	ProcessFactory() = default;
	virtual ~ProcessFactory() = default;
};
