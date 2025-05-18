#pragma once

#include "io/IOFactory.hpp"

/**
 * Unix-specific implementation of the IOFactory
 */
class UnixIOFactory final : public IOFactory
{
public:
	/**
	 * Constructor
	 */
	UnixIOFactory() = default;

	/**
	 * Create a file sink for Unix systems
	 * @param path Path to the file
	 * @param append Whether to append to existing file
	 * @return Shared pointer to ISink interface
	 */
	std::shared_ptr<ISink> createFileSink(const std::filesystem::path &path, bool append = false) override;

	/**
	 * Create a file source for Unix systems
	 * @param path Path to the file
	 * @return Shared pointer to ISource interface
	 */
	std::shared_ptr<ISource> createFileSource(const std::filesystem::path &path) override;

	/**
	 * Create a pipe for Unix systems
	 * @return Shared pointer to IPipe interface
	 */
	std::shared_ptr<IPipe> createPipe() override;
};
