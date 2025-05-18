#pragma once

#include "ISource.hpp"

// File descriptor implementation of ISource
class FDSource final : public ISource
{
public:
	// Create from an existing file descriptor
	explicit FDSource(NativeHandle fd);
	~FDSource() override;

	size_t read(void *buffer, size_t size) override;
	bool canRead() const override;
	void close() override;
	NativeHandle getNativeHandle() const override { return fd; }

private:
	NativeHandle fd;
	bool closed = false;
};