cmake_minimum_required(VERSION 3.12)

# Set compiler to Homebrew Clang for macOS
if(APPLE)
  set(CMAKE_C_COMPILER "/opt/homebrew/bin/clang")
  set(CMAKE_CXX_COMPILER "/opt/homebrew/bin/clang++")
endif()

project(foundation_shell VERSION 1.0)

# Add a custom command to check for incorrect static_assert usage
add_custom_target(check_static_asserts
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/fix_static_asserts.sh
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for incorrect static_assert usage..."
)

# Add a custom command to check for proper upper CamelCase (PascalCase) file naming
add_custom_target(check_casing
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check_casing.ps1
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for proper upper CamelCase (PascalCase) file naming..."
)

# Add a custom command to check for proper include paths
add_custom_target(check_includes
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check_includes.ps1
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for proper include paths..."
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

# Set up mh-cmake-common
# Make the package findable
set(CMAKE_PREFIX_PATH "${CMAKE_SOURCE_DIR}/extern/vcpkg-registry/ports")
list(APPEND CMAKE_MODULE_PATH "${CMAKE_SOURCE_DIR}/extern/vcpkg-registry/ports/mh-cmake-common")

# Add MH stuff submodule
add_subdirectory(extern/mh_stuff)

# Add platform implementation library
add_subdirectory(platform_impl)

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

# Create library with all sources
add_library(foundation_shell_lib ${SOURCES})
target_compile_features(foundation_shell_lib PUBLIC cxx_std_23)

# Link against dependencies
target_link_libraries(foundation_shell_lib PUBLIC mh-stuff platform_impl)

# Enable coroutine support
if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
  # GCC and Clang already have coroutine support in C++20/C++23
endif()

# Set include directories for library
target_include_directories(foundation_shell_lib PUBLIC src)

# Main executable
add_executable(foundation_shell
    src/main.cpp
)

# Link executable against the library
target_link_libraries(foundation_shell PRIVATE foundation_shell_lib)

# Make sure all checks run before building
add_dependencies(foundation_shell check_static_asserts check_casing check_includes)

# Enable testing
enable_testing()

# Find all test files
file(GLOB_RECURSE TEST_FILES CONFIGURE_DEPENDS
    "tests/*_test.cpp"
)

# Create a single test executable for all tests
add_executable(unit_tests ${TEST_FILES})
target_link_libraries(unit_tests
    foundation_shell_lib
    gtest_main
    gmock
)

# Add the test to CTest
add_test(NAME UnitTests COMMAND unit_tests)
