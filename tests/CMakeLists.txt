# Enable testing
enable_testing()

# Find all test files
file(GLOB_RECURSE TEST_FILES CONFIGURE_DEPENDS
    "*_test.cpp"
    "test_utils.cpp"
)

# Create a single test executable for all tests
add_executable(unit_tests ${TEST_FILES})
target_link_libraries(unit_tests
    foundation_shell_lib
    Catch2::Catch2WithMain
)

# Add include directories for test files (using absolute paths)
target_include_directories(unit_tests PRIVATE
    "${CMAKE_SOURCE_DIR}"
)

include(CTest)
list(APPEND CMAKE_MODULE_PATH ${Catch2_SOURCE_DIR}/extras)
include(Catch)
catch_discover_tests(unit_tests)
