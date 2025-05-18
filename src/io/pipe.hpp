#pragma once

#include "IPipe.hpp"
#include <memory>

// Platform-independent function to create a pipe
// Returns a shared_ptr to an IPipe
std::shared_ptr<IPipe> createPipe();