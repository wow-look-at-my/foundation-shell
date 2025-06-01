#pragma once

#include <filesystem>
#include <memory>

#include "ISource.hpp"

/**
 * Platform-agnostic file source
 * Acts as a facade for platform-specific implementations
 */
class FileSource
{
public:
	/**
	 * Create a file source
	 * @param path Path to the file
	 * @return Shared pointer to an ISource implementation
	 */
	static std::shared_ptr<ISource> create(const std::filesystem::path& path);
};

// Typedef for shared pointer to ISource
using FileSourcePtr = std::shared_ptr<ISource>;
