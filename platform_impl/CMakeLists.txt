# Platform implementations CMake file
cmake_minimum_required(VERSION 3.12)

# Find Unix implementation files
file(GLOB_RECURSE UNIX_SOURCES
    "${CMAKE_CURRENT_SOURCE_DIR}/unix/io/*.cpp"
    "${CMAKE_CURRENT_SOURCE_DIR}/unix/process/*.cpp"
)

# Create Unix implementation library
add_library(platform_impl STATIC ${UNIX_SOURCES})
target_compile_features(platform_impl PUBLIC cxx_std_23)

# Link with core and mh-stuff to get access to global dispatcher and task.hpp
target_link_libraries(platform_impl PRIVATE core mh-stuff)

# Include directories for finding header files
# This is internal - only for building this library
target_include_directories(platform_impl
    PRIVATE
		"${CMAKE_SOURCE_DIR}"
		"${CMAKE_CURRENT_SOURCE_DIR}"
		"${CMAKE_SOURCE_DIR}/src"
)
