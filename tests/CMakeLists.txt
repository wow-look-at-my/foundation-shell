
# Only create tests if BUILD_TESTING is enabled and Catch2 is available
if(BUILD_TESTING)

    # Enable testing
    enable_testing()

    # Find all test files
    file(GLOB_RECURSE TEST_FILES CONFIGURE_DEPENDS
        "*Test.cpp"
        "TestUtils.cpp"
        "TestMain.cpp"
    )

    include(CTest)

    # Create a single test executable for all tests
    add_executable(unit_tests ${TEST_FILES})
    target_link_libraries(unit_tests
        foundation_shell_lib
        Catch2::Catch2
    )

    # Add include directories for test files (using absolute paths)
    target_include_directories(unit_tests PRIVATE
        "${CMAKE_SOURCE_DIR}"
    )

    catch_discover_tests(unit_tests)
endif()
