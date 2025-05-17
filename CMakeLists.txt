cmake_minimum_required(VERSION 3.12)
project(foundation_shell VERSION 1.0)

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
add_executable(foundation_shell
    src/main.cpp
    src/config.cpp
    src/command.cpp
    src/history.cpp
    src/job.cpp
    src/alias.cpp
)

# Set include directory
target_include_directories(foundation_shell PRIVATE include)

# Enable testing
enable_testing()

# Add test executables
add_executable(shell_test tests/shell_test.cpp)
target_link_libraries(shell_test gtest_main gmock)
target_include_directories(shell_test PRIVATE include)

add_executable(features_test tests/features_test.cpp)
target_link_libraries(features_test gtest_main gmock)
target_include_directories(features_test PRIVATE include)

# Add tests to CTest
add_test(NAME ShellTest COMMAND shell_test)
add_test(NAME FeaturesTest COMMAND features_test)
