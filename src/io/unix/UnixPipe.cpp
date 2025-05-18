#include "../../include/io/UnixPipe.hpp"
#include <unistd.h>
#include <stdexcept>
#include <memory>

UnixPipe::UnixPipe()
{
    int pipefd[2];
    if (pipe(pipefd) == -1) {
        throw std::runtime_error("Failed to create pipe");
    }
    
    // Create FDSource for read end (pipefd[0]) and FDSink for write end (pipefd[1])
    // The FDSource and FDSink will own and close their respective file descriptors
    source = std::make_shared<FDSource>(pipefd[0]);
    sink = std::make_shared<FDSink>(pipefd[1]);
}

std::shared_ptr<ISource> UnixPipe::getSource() const
{
    return source;
}

std::shared_ptr<ISink> UnixPipe::getSink() const
{
    return sink;
}