#include "UnixFileSource.hpp"
#include <fcntl.h>
#include <unistd.h>
#include <stdexcept>

UnixFileSource::UnixFileSource(const std::filesystem::path &path)
	: FDSource(::open(path.c_str(), O_RDONLY))
{
	if (getNativeHandle() == -1)
	{
		throw std::runtime_error("Failed to open file for reading: " + path.string());
	}
}
