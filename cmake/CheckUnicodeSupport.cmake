# Check if the current compiler has broken Unicode support
# Tests if the compiler can properly use mbrtoc32 and other Unicode conversion functions

include(CheckCXXSourceCompiles)

# Test code that uses Unicode conversion functions from <cuchar>
set(UNICODE_TEST_CODE "
#include <cuchar>
#include <cstddef>

int main() {
  char32_t c32;
  char s[5] = \"test\";
  std::mbstate_t ps{};
  
  // Try to call mbrtoc32 function - one of the problematic functions
  std::size_t result = std::mbrtoc32(&c32, s, 5, &ps);
  
  return (result == static_cast<std::size_t>(-1)) ? 1 : 0;
}
")

# Try to compile the test code
check_cxx_source_compiles("${UNICODE_TEST_CODE}" UNICODE_SUPPORT_WORKS)

# If compilation fails, define MH_BROKEN_UNICODE
if(NOT UNICODE_SUPPORT_WORKS)
  message(STATUS "Detected broken Unicode support - enabling MH_BROKEN_UNICODE workaround")
  add_compile_definitions(MH_BROKEN_UNICODE=1)
else()
  message(STATUS "Unicode support works correctly")
endif()