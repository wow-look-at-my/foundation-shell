#include "GlobalDispatcher.hpp"

namespace core {
    mh::dispatcher& getGlobalDispatcher() {
        static mh::dispatcher instance;
        return instance;
    }
}