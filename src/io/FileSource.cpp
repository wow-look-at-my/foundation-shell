#include "FileSource.hpp"

#include "IOFactory.hpp"
#include "LastCppInclude.hpp"

std::shared_ptr<ISource> FileSource::create(const std::filesystem::path& path)
{
	// Use the factory to create the appropriate platform-specific implementation
	return IOFactory::getInstance().createFileSource(path);
}
