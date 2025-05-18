#pragma once

#include "ISink.hpp"

// File descriptor implementation of ISink
class UnixFDSink final : public ISink
{
public:
	// Create from an existing file descriptor
	explicit UnixFDSink(NativeHandle fd);
	~UnixFDSink() override;

	size_t write(const void *buffer, size_t size) override;
	bool canWrite() const override;
	void flush() override;
	void close() override;
	NativeHandle getNativeHandle() const override { return fd; }

private:
	NativeHandle fd;
	bool closed = false;
};
