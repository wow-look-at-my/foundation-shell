#pragma once

#include <filesystem>
#include <memory>

#include "ISink.hpp"

/**
 * Platform-agnostic file sink
 * Acts as a facade for platform-specific implementations
 */
class FileSink
{
public:
	/**
	 * Create a file sink
	 * @param path Path to the file
	 * @param append Whether to append to the file if it exists
	 * @return Shared pointer to an ISink implementation
	 */
	static std::shared_ptr<ISink> create(const std::filesystem::path& path, bool append = false);
};

// Typedef for shared pointer to ISink
using FileSinkPtr = std::shared_ptr<ISink>;
