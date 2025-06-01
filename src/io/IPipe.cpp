#include "IPipe.hpp"

#include "IOFactory.hpp"
#include "LastCppInclude.hpp"

std::shared_ptr<IPipe> IPipe::create()
{
	// Use the factory to create the appropriate platform-specific implementation
	return IOFactory::getInstance().createPipe();
}
