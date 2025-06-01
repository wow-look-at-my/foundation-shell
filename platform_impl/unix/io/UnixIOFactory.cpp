#include "UnixIOFactory.hpp"
#include "UnixFileSink.hpp"
#include "UnixFileSource.hpp"
#include "UnixPipe.hpp"
#include "LastCppInclude.hpp"

// Implementation of the singleton getInstance method
IOFactory &IOFactory::getInstance()
{
	// Create the singleton instance of UnixIOFactory
	static UnixIOFactory instance;
	return instance;
}

// Implementation of UnixIOFactory methods
std::shared_ptr<ISink> UnixIOFactory::createFileSink(const std::filesystem::path &path, bool append)
{
	return std::make_shared<UnixFileSink>(path, append);
}

std::shared_ptr<ISource> UnixIOFactory::createFileSource(const std::filesystem::path &path)
{
	return std::make_shared<UnixFileSource>(path);
}

std::shared_ptr<IPipe> UnixIOFactory::createPipe()
{
	return std::make_shared<UnixPipe>();
}
