#include "FileSink.hpp"
#include "IOFactory.hpp"
#include "LastCppInclude.hpp"

std::shared_ptr<ISink> FileSink::create(const std::filesystem::path &path, bool append)
{
	// Use the factory to create the appropriate platform-specific implementation
	return IOFactory::getInstance().createFileSink(path, append);
}
