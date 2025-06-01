#pragma once

#include "src/io/ISource.hpp"
#include <cstdint>

// Forward declaration
class FDSource;

// Typedef for shared pointer to FDSource
using FDSourcePtr = std::shared_ptr<FDSource>;

/**
 * Implementation of ISource that reads from a file descriptor
 * Platform-specific implementation for Unix-like systems
 */
class FDSource : public ISource
{
public:
	// Native file descriptor type (must match ISource::NativeHandle)

	/**
	 * Constructor that takes ownership of a file descriptor
	 */
	explicit FDSource(NativeHandle fd);

	/**
	 * Destructor that closes the file descriptor if still open
	 */
	~FDSource() override;

	/**
	 * Read data from the file descriptor
	 * @param buffer Buffer to read into
	 * @param size Maximum number of bytes to read
	 * @return Actual number of bytes read, 0 on EOF or error
	 */
	size_t read(void *buffer, size_t size) override;

	/**
	 * Check if the source can be read from without blocking
	 * @return true if read would not block, false otherwise
	 */
	bool canRead() const override;

	/**
	 * Close the file descriptor
	 */
	void close() override;

	/**
	 * Get the native file descriptor handle
	 * @return The file descriptor as NativeHandle (intptr_t)
	 */
	NativeHandle getNativeHandle() const override { return static_cast<NativeHandle>(fd); }

private:
	int fd;		 // The file descriptor
	bool closed; // Whether the file descriptor has been closed
};
