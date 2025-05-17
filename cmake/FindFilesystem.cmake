# Distributed under the OSI-approved BSD 3-Clause License.
# Copyright (c) 2019, Marius Brehler

#[=======================================================================[.rst:
FindFilesystem
--------------

This module supports the C++17 standard library's filesystem API.

.. versionadded:: 3.12

The following variables are set:

``Filesystem_FOUND``
  True if the Filesystem library is found.
``Filesystem_INCLUDE_DIRS``
  Include directory needed for Filesystem.
``Filesystem_LIBRARIES``
  Libraries needed to link to Filesystem.

Cache variables
^^^^^^^^^^^^^^

``Filesystem_LIBRARY``
  The path to the filesystem library.
``Filesystem_INCLUDE_DIR``
  The directory containing the filesystem headers.

#]=======================================================================]

include(CMakePushCheckState)
include(CheckIncludeFileCXX)

# Allow users to specify a specific path to the filesystem library
if(NOT Filesystem_ROOT)
  set(Filesystem_ROOT "" CACHE PATH "Preferred installation prefix of the filesystem library.")
endif()

# Set up the search paths
if(Filesystem_ROOT)
  set(_Filesystem_INCLUDE_DIR_HINT ${Filesystem_ROOT}/include)
  set(_Filesystem_LIBRARY_DIR_HINT ${Filesystem_ROOT}/lib)
endif()

# Check for header file
cmake_push_check_state(RESET)
set(CMAKE_REQUIRED_QUIET ${Filesystem_FIND_QUIETLY})

# Detect which filesystem implementation to use
set(HEADERS_TO_CHECK
    "filesystem"
    "experimental/filesystem")

set(filesystem_header_found FALSE)
set(filesystem_namespace "filesystem")
foreach(header ${HEADERS_TO_CHECK})
  check_include_file_cxx("${header}" have_header)
  if(have_header)
    set(filesystem_header "${header}")
    set(filesystem_header_found TRUE)
    if(header STREQUAL "experimental/filesystem")
      set(filesystem_namespace "experimental::filesystem")
    endif()
    break()
  endif()
endforeach()

# If we found a filesystem header, try to compile and run a simple program
if(filesystem_header_found)
  set(CMAKE_REQUIRED_FLAGS "${CMAKE_CXX_STANDARD_REQUIRED_FLAG}")
  set(CMAKE_REQUIRED_LIBRARIES "-lstdc++fs" "-lc++fs")

  # Try to compile a simple program to check for functionality
  check_cxx_source_compiles("
    #include <${filesystem_header}>
    int main() {
      auto path = std::${filesystem_namespace}::path(\"\");
      return 0;
    }
  " Filesystem_FOUND)
endif()

# Restore original state
cmake_pop_check_state()

# Find the actual library
if(Filesystem_FOUND)
  # Check if we need to link against a library for filesystem
  set(CMAKE_REQUIRED_FLAGS "${CMAKE_CXX_STANDARD_REQUIRED_FLAG}")
  check_cxx_source_compiles("
    #include <${filesystem_header}>
    int main() {
      auto path = std::${filesystem_namespace}::path(\"\");
      return 0;
    }
  " Filesystem_NO_LIBRARY_NEEDED)

  # If we need to link against a library, try to find it
  if(NOT Filesystem_NO_LIBRARY_NEEDED)
    # Try standard libraries first
    if(CMAKE_CXX_COMPILER_ID MATCHES "GNU")
      find_library(Filesystem_LIBRARY
        NAMES stdc++fs
        HINTS ${_Filesystem_LIBRARY_DIR_HINT})
      if(Filesystem_LIBRARY)
        set(Filesystem_LIBRARIES ${Filesystem_LIBRARY})
      endif()
    elseif(CMAKE_CXX_COMPILER_ID MATCHES "Clang")
      find_library(Filesystem_LIBRARY
        NAMES c++fs
        HINTS ${_Filesystem_LIBRARY_DIR_HINT})
      if(Filesystem_LIBRARY)
        set(Filesystem_LIBRARIES ${Filesystem_LIBRARY})
      endif()
    endif()
  endif()
endif()

# Handle results
include(FindPackageHandleStandardArgs)
find_package_handle_standard_args(Filesystem
  REQUIRED_VARS Filesystem_FOUND)

if(Filesystem_FOUND AND NOT TARGET Filesystem::Filesystem)
  add_library(Filesystem::Filesystem INTERFACE IMPORTED)
  if(Filesystem_LIBRARIES)
    set_target_properties(Filesystem::Filesystem PROPERTIES
      INTERFACE_LINK_LIBRARIES "${Filesystem_LIBRARIES}")
  endif()
endif()
