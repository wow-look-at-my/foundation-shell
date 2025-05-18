#include "UnixProcessFactory.hpp"
#include "UnixProcess.hpp"

// Implementation of the singleton getInstance method
ProcessFactory &ProcessFactory::getInstance()
{
	// Create the singleton instance of UnixProcessFactory
	static UnixProcessFactory instance;
	return instance;
}

// Implementation of UnixProcessFactory methods
std::shared_ptr<IProcess> UnixProcessFactory::createProcess(
	const std::string &command,
	const std::vector<std::string> &args,
	Source inputSource,
	Sink outputSink,
	Sink errorSink)
{
	return std::make_shared<UnixProcess>(command, args, inputSource, outputSink, errorSink);
}
