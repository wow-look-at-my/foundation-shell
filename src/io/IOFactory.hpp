#pragma once

#include <filesystem>
#include <memory>

#include "IPipe.hpp"
#include "ISink.hpp"
#include "ISource.hpp"

/**
 * Factory class for creating IO objects
 * This abstract factory allows platform-specific implementations to be created
 * without direct dependencies on platform-specific code
 */
class IOFactory
{
public:
	/**
	 * Get the singleton instance of the factory
	 */
	static IOFactory& getInstance();

	/**
	 * Create a file sink
	 * @param path Path to the file
	 * @param append Whether to append to existing file
	 * @return Shared pointer to ISink interface
	 */
	virtual std::shared_ptr<ISink> createFileSink(const std::filesystem::path& path, bool append = false) = 0;

	/**
	 * Create a file source
	 * @param path Path to the file
	 * @return Shared pointer to ISource interface
	 */
	virtual std::shared_ptr<ISource> createFileSource(const std::filesystem::path& path) = 0;

	/**
	 * Create a pipe for interprocess communication
	 * @return Shared pointer to IPipe interface
	 */
	virtual std::shared_ptr<IPipe> createPipe() = 0;

protected:
	// Protected constructor for singleton pattern
	IOFactory() = default;
	virtual ~IOFactory() = default;
};
