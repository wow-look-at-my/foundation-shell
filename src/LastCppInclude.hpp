#pragma once

#pragma GCC poison cout cerr cin


// No std::string::find/contains in tests, use Catch2 matchers instead.
#ifdef CATCH_CONFIG_HPP_INCLUDED
#pragma GCC poison find contains
#endif
