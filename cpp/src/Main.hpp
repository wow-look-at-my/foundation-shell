#pragma once

#include <chrono>

#include "mh/coroutine/task.hpp"

extern mh::task<void> sleep_async(std::chrono::milliseconds duration);
