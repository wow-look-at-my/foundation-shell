#pragma once

#include "mh/coroutine/task.hpp"
#include <chrono>

extern mh::task<void> sleep_async(std::chrono::milliseconds duration);
