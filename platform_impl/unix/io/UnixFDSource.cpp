#include "FDSource.hpp"
#include <unistd.h>
#include <fcntl.h>
#include <sys/select.h>

FDSource::FDSource(NativeHandle fd)
    : fd(fd), closed(false)
{
}

FDSource::~FDSource()
{
    if (!closed) {
        close();
    }
}

size_t FDSource::read(void *buffer, size_t size)
{
    if (closed) {
        return 0;
    }
    
    ssize_t bytesRead = ::read(fd, buffer, size);
    return bytesRead >= 0 ? static_cast<size_t>(bytesRead) : 0;
}

bool FDSource::canRead() const
{
    if (closed) {
        return false;
    }
    
    // Check if file descriptor is readable using select
    fd_set readfds;
    FD_ZERO(&readfds);
    FD_SET(fd, &readfds);
    
    // Zero timeout for immediate return
    struct timeval timeout;
    timeout.tv_sec = 0;
    timeout.tv_usec = 0;
    
    return select(fd + 1, &readfds, nullptr, nullptr, &timeout) > 0;
}

void FDSource::close()
{
    if (!closed) {
        ::close(fd);
        closed = true;
    }
}