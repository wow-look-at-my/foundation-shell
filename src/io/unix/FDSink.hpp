#pragma once

#include "ISink.hpp"
#include <cstdint>

// Forward declaration
class FDSink;

// Typedef for shared pointer to FDSink
using FDSinkPtr = std::shared_ptr<FDSink>;

/**
 * Implementation of ISink that writes to a file descriptor
 * Platform-specific implementation for Unix-like systems
 */
class FDSink : public ISink {
public:
    // Type definition for native file descriptor handle
    using NativeHandle = int;
    
    /**
     * Constructor that takes ownership of a file descriptor
     */
    explicit FDSink(NativeHandle fd);
    
    /**
     * Destructor that closes the file descriptor if still open
     */
    ~FDSink() override;
    
    /**
     * Write data to the file descriptor
     * @param buffer Buffer to write from
     * @param size Number of bytes to write
     * @return Actual number of bytes written, 0 on error
     */
    size_t write(const void* buffer, size_t size) override;
    
    /**
     * Check if the sink can be written to without blocking
     * @return true if write would not block, false otherwise
     */
    bool canWrite() const override;
    
    /**
     * Flush any buffered data to ensure it is written
     */
    void flush() override;
    
    /**
     * Close the file descriptor
     */
    void close() override;
    
    /**
     * Get the native file descriptor handle
     * @return The file descriptor
     */
    NativeHandle getNativeHandle() const { return fd; }

private:
    NativeHandle fd;  // The file descriptor
    bool closed;      // Whether the file descriptor has been closed
};