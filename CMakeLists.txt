cmake_minimum_required(VERSION 3.10)
project(base_shell VERSION 0.1)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

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

# Add test executable
add_executable(shell_test tests/shell_test.cpp)
target_link_libraries(shell_test gtest_main gmock)

# Add test to CTest
add_test(NAME ShellTest COMMAND shell_test)
