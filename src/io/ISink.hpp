#pragma once

#include <cstddef>

// Platform-agnostic abstract handle type
using NativeHandle = intptr_t;

// Interface for data sinks (e.g., stdin)
class ISink
{
public:
	virtual ~ISink() = default;

	// Write data to sink
	virtual size_t write(const void *buffer, size_t size) = 0;

	// Check if sink can accept data
	virtual bool canWrite() const = 0;

	// Flush any buffered data
	virtual void flush() = 0;

	// Close the sink
	virtual void close() = 0;

	// Get native handle (file descriptor on Unix, HANDLE on Windows)
	virtual NativeHandle getNativeHandle() const = 0;
};