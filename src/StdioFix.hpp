#pragma once

// This header must be included before any STL headers that use print
// to ensure stdout and other stdio identifiers are properly defined

// Save the current macro state
#pragma push_macro("stdin")
#pragma push_macro("stdout")
#pragma push_macro("stderr")

// Undefine macros in case they were already defined
#ifdef stdin
#undef stdin
#endif

#ifdef stdout
#undef stdout
#endif

#ifdef stderr
#undef stderr
#endif

// Include cstdio to get proper definitions
#include <cstdio>

// Restore the original macro state
#pragma pop_macro("stderr")
#pragma pop_macro("stdout")
#pragma pop_macro("stdin")
