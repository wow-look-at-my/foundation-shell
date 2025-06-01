#pragma once

#include <cstddef>
#include <memory>

// Platform-agnostic abstract handle type
using NativeHandle = intptr_t;

// Interface for data sources (e.g., stdout, stderr)
class ISource
{
public:
	virtual ~ISource() = default;

	// Read data from source
	virtual size_t read(void* buffer, size_t size) = 0;

	// Check if there's data available to read
	virtual bool canRead() const = 0;

	// Close the source
	virtual void close() = 0;

	// Get native handle (file descriptor on Unix, HANDLE on Windows)
	virtual NativeHandle getNativeHandle() const = 0;
};

// Shorthand for shared pointer to ISource
using Source = std::shared_ptr<ISource>;