cmake_minimum_required(VERSION 3.12)

# Enable compile_commands.json generation for all projects
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

# Set compiler to Homebrew Clang for macOS
if(APPLE)
  set(CMAKE_C_COMPILER "/opt/homebrew/bin/clang")
  set(CMAKE_CXX_COMPILER "/opt/homebrew/bin/clang++")
endif()

project(foundation_shell VERSION 1.0)

# Enable warnings as errors and colored output
if(CMAKE_CXX_COMPILER_ID MATCHES "Clang" OR CMAKE_CXX_COMPILER_ID MATCHES "GNU")
    add_compile_options(-Werror -fdiagnostics-color=always)
endif()

# Add a custom command to check for incorrect static_assert usage
add_custom_target(check_static_asserts
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/fix_static_asserts.cs
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for incorrect static_assert usage..."
)

# Add a custom command to check for proper upper CamelCase (PascalCase) file naming
add_custom_target(check_casing
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check_casing.cs
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for proper upper CamelCase (PascalCase) file naming..."
)

# Add a custom command to check for proper include paths
add_custom_target(check_includes
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check_includes.cs ${CMAKE_SOURCE_DIR}
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking for proper include paths..."
)

# Add a custom command to check that LastCppInclude.hpp is the last include in all cpp files
add_custom_target(check_last_include
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check_last_include.cs
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
  COMMENT "Checking that LastCppInclude.hpp is the last include in all cpp files..."
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

# Add Catch2 only if testing is enabled
if(BUILD_TESTING)
  CPMAddPackage(
    NAME Catch2
    GITHUB_REPOSITORY catchorg/Catch2
    VERSION 3.4.0
    GIT_TAG v3.4.0
  )

  # Disable deprecated literal operator warning for Catch2
  if(TARGET Catch2)
    target_compile_options(Catch2 PRIVATE -Wno-deprecated-literal-operator)
  endif()

  # Make Catch2 CMake extensions available
  list(APPEND CMAKE_MODULE_PATH ${catch2_SOURCE_DIR}/extras)
endif()

# Add MH stuff submodule
add_subdirectory(extern/mh_stuff)

# Add core library (shared between main and platform implementations)
add_subdirectory(src/core)

# Add JSON library (header-only)
CPMAddPackage(
  NAME nlohmann_json
  GITHUB_REPOSITORY nlohmann/json
  GIT_TAG v3.12.0
  OPTIONS "JSON_BuildTests OFF"
)

# Enable policy to correctly handle CONFIGURE_DEPENDS option
cmake_policy(SET CMP0098 NEW)

# Use glob_recurse to find all source files with CONFIGURE_DEPENDS to properly handle dependencies
file(GLOB_RECURSE SOURCES CONFIGURE_DEPENDS
    "src/*.cpp"
)

# Remove main.cpp from the sources as it will be added directly to the executable
list(FILTER SOURCES EXCLUDE REGEX ".*[mM]ain\\.cpp$")

# Create library with all sources
add_library(foundation_shell_lib ${SOURCES})
target_compile_features(foundation_shell_lib PUBLIC cxx_std_23)

# Link against dependencies
target_link_libraries(foundation_shell_lib PUBLIC core mh-stuff nlohmann_json)

# Set include directories for library
target_include_directories(foundation_shell_lib PUBLIC src)

# Main executable
add_executable(foundation_shell
    src/Main.cpp
)

target_compile_definitions(foundation_shell PRIVATE FOUNDATION_SHELL_EXE)

# Link executable against the library
target_link_libraries(foundation_shell PRIVATE foundation_shell_lib)

# Make sure all checks run before building
add_dependencies(foundation_shell check_static_asserts check_casing check_includes check_last_include)

add_subdirectory(tests)
