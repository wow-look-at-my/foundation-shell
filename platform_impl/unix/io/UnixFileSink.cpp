#include "UnixFileSink.hpp"
#include <fcntl.h>
#include <unistd.h>
#include <stdexcept>
#include "LastCppInclude.hpp"

UnixFileSink::UnixFileSink(const std::filesystem::path &path, bool append)
	: FDSink(::open(path.c_str(),
					append ? (O_WRONLY | O_CREAT | O_APPEND) : (O_WRONLY | O_CREAT | O_TRUNC),
					0644))
{
	if (getNativeHandle() == -1)
	{
		throw std::runtime_error("Failed to open file for writing: " + path.string());
	}
}
