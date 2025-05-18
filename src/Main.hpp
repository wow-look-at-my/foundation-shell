#pragma once

#include "Task.hpp"
#include <chrono>

extern Task<void> sleep_async(std::chrono::milliseconds duration);
