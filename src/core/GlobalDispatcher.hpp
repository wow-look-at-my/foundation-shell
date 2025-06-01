#pragma once

#include <mh/concurrency/dispatcher.hpp>

namespace core
{
// Global dispatcher singleton - one per program
mh::dispatcher& getGlobalDispatcher();
} // namespace core