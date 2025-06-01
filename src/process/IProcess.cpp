#include "IProcess.hpp"
#include "ProcessFactory.hpp"
#include "LastCppInclude.hpp"

ProcessPtr IProcess::create(const ProcessOptions &options)
{
	// Use the factory to create the appropriate platform-specific implementation
	return ProcessFactory::getInstance().createProcess(
		options.command,
		options.args,
		options.inputSource,
		options.outputSink,
		options.errorSink);
}
