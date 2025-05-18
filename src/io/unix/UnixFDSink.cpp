#include "../../include/io/FDSink.hpp"
#include <unistd.h>
#include <fcntl.h>
#include <sys/select.h>

FDSink::FDSink(NativeHandle fd)
    : fd(fd), closed(false)
{
}

FDSink::~FDSink()
{
    if (!closed) {
        close();
    }
}

size_t FDSink::write(const void *buffer, size_t size)
{
    if (closed) {
        return 0;
    }
    
    ssize_t bytesWritten = ::write(fd, buffer, size);
    return bytesWritten >= 0 ? static_cast<size_t>(bytesWritten) : 0;
}

bool FDSink::canWrite() const
{
    if (closed) {
        return false;
    }
    
    // Check if file descriptor is writable using select
    fd_set writefds;
    FD_ZERO(&writefds);
    FD_SET(fd, &writefds);
    
    // Zero timeout for immediate return
    struct timeval timeout;
    timeout.tv_sec = 0;
    timeout.tv_usec = 0;
    
    return select(fd + 1, nullptr, &writefds, nullptr, &timeout) > 0;
}

void FDSink::flush()
{
    if (!closed) {
        fsync(fd);
    }
}

void FDSink::close()
{
    if (!closed) {
        ::close(fd);
        closed = true;
    }
}