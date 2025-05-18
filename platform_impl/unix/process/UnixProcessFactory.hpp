#pragma once

#include "process/ProcessFactory.hpp"

/**
 * Unix-specific implementation of the ProcessFactory
 */
class UnixProcessFactory final : public ProcessFactory
{
public:
	/**
	 * Constructor
	 */
	UnixProcessFactory() = default;

	/**
	 * Create a process for Unix systems
	 * @param command The command to execute
	 * @param args Command arguments
	 * @param inputSource Input source for the process (stdin)
	 * @param outputSink Output sink for the process (stdout)
	 * @param errorSink Error sink for the process (stderr)
	 * @return Shared pointer to IProcess interface
	 */
	std::shared_ptr<IProcess> createProcess(
		const std::string &command,
		const std::vector<std::string> &args,
		Source inputSource,
		Sink outputSink,
		Sink errorSink) override;
};
