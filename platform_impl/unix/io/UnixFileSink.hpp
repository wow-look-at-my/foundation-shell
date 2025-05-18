#pragma once

#include "FDSink.hpp"
#include <filesystem>

// File-based implementation of ISink that wraps an FDSink
class UnixFileSink final : public FDSink
{
public:
	// Open a file for writing using filesystem path
	explicit UnixFileSink(const std::filesystem::path &path, bool append = false);
};
