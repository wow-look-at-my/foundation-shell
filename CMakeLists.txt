cmake_minimum_required(VERSION 3.12)
project(base_shell VERSION 0.1)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

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

# Main executable
add_executable(base_shell src/main.cpp)

# Enable testing
enable_testing()

# Add test executables
add_executable(shell_test tests/shell_test.cpp)
target_link_libraries(shell_test gtest_main gmock)

add_executable(features_test tests/features_test.cpp)
target_link_libraries(features_test gtest_main gmock)

# Add tests to CTest
add_test(NAME ShellTest COMMAND shell_test)
add_test(NAME FeaturesTest COMMAND features_test)
