#include "../../../include/io/pipe.hpp"
#include "../../../include/io/UnixPipe.hpp"
#include <memory>

// Platform-independent function to create a pipe
// This is the Unix implementation
std::shared_ptr<IPipe> createPipe()
{
    // On Unix platforms, create a UnixPipe
    return std::make_shared<UnixPipe>();
}