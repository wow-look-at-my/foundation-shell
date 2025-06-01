#pragma once

#include "src/process/IProcess.hpp"
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <fcntl.h>

/**
 * Unix implementation of the IProcess interface
 */
class UnixProcess : public IProcess
{
public:
	/**
	 * Constructor
	 * @param command The command to execute
	 * @param args Command arguments
	 * @param inputSource Input source for the process (stdin)
	 * @param outputSink Output sink for the process (stdout)
	 * @param errorSink Error sink for the process (stderr)
	 */
	UnixProcess(
		const std::string &command,
		const std::vector<std::string> &args,
		Source inputSource = nullptr,
		Sink outputSink = nullptr,
		Sink errorSink = nullptr);

	/**
	 * Destructor
	 */
	~UnixProcess() override;

	/**
	 * Start the process
	 * @return True if process started successfully
	 */
	bool start() override;

	/**
	 * Wait for the process to complete
	 * @return Exit code of the process
	 */
	Task<int> waitAsync() override;

	/**
	 * Check if the process is running
	 * @return True if process is still running
	 */
	bool isRunning() const override;

	/**
	 * Get the process ID
	 * @return Unix process ID
	 */
	intptr_t getPid() const override;

	/**
	 * Terminate the process
	 */
	void terminate() override;

private:
	std::string command_;
	std::vector<std::string> args_;
	Source inputSource_;
	Sink outputSink_;
	Sink errorSink_;
	pid_t pid_;
	bool started_;
	bool completed_;
	int exitCode_;

	// Convert vector of strings to array of C-strings
	char **vectorToCharArray(const std::vector<std::string> &args) const;

	// Free memory allocated for char array
	void freeCharArray(char **array, int size) const;

	// Setup I/O redirection for child process
	void setupChildIO() const;
};
