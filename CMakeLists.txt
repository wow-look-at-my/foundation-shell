cmake_minimum_required(VERSION 3.12)
project(foundation_shell VERSION 1.0)

# Add a custom command to check for incorrect static_assert usage
add_custom_target(check_static_asserts
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/fix_static_asserts.sh
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for incorrect static_assert usage..."
)

# Check for filesystem support
if(APPLE)
  # MacOS has built-in filesystem support without a separate library
  # Only define a preprocessor macro to identify we're on macOS
  add_compile_definitions(FILESYSTEM_APPLE)
else()
  # For Linux and other systems, try to find filesystem library
  find_package(Filesystem QUIET)
  if(NOT Filesystem_FOUND)
    # Fallback to compiler-specific filesystem libraries if needed
    if(CMAKE_CXX_COMPILER_ID MATCHES "GNU")
      link_libraries(stdc++fs)
    elseif(CMAKE_CXX_COMPILER_ID MATCHES "Clang" AND NOT APPLE)
      link_libraries(c++fs)
    endif()
  endif()
endif()

include("cmake/CPM.cmake")

# Add GoogleTest using CPM
CPMAddPackage(
  NAME googletest
  GITHUB_REPOSITORY google/googletest
  GIT_TAG v1.14.0
  OPTIONS "INSTALL_GTEST OFF" "gtest_force_shared_crt ON"
)

# Enable policy to correctly handle CONFIGURE_DEPENDS option
cmake_policy(SET CMP0098 NEW)

# Use glob_recurse to find all source files with CONFIGURE_DEPENDS to properly handle dependencies
file(GLOB_RECURSE SOURCES CONFIGURE_DEPENDS
    "src/*.cpp"
)

# Remove main.cpp from the sources as it will be added directly to the executable
list(FILTER SOURCES EXCLUDE REGEX ".*main\\.cpp$")

# Main executable
add_executable(foundation_shell
    src/main.cpp
    ${SOURCES}
)

# Make sure the static_assert check runs before building
add_dependencies(foundation_shell check_static_asserts)

target_compile_features(foundation_shell PUBLIC cxx_std_23)

# Set include directories
target_include_directories(foundation_shell PRIVATE src)

# Enable testing
enable_testing()

# Find all test files
file(GLOB_RECURSE TEST_FILES CONFIGURE_DEPENDS
    "tests/*_test.cpp"
)

# Create a single test executable for all tests
add_executable(unit_tests ${TEST_FILES})
target_link_libraries(unit_tests gtest_main gmock)
target_include_directories(unit_tests PRIVATE src)

# Add the test to CTest
add_test(NAME UnitTests COMMAND unit_tests)
