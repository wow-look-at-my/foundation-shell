#include "../../include/io/FileSource.hpp"
#include <fcntl.h>
#include <unistd.h>
#include <stdexcept>

FileSource::FileSource(const std::filesystem::path& path)
    : FDSource(::open(path.c_str(), O_RDONLY))
{
    if (getNativeHandle() == -1) {
        throw std::runtime_error("Failed to open file for reading: " + path.string());
    }
}