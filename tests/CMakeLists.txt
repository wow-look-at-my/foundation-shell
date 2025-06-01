# Enable testing
enable_testing()

# Find all test files
file(GLOB_RECURSE TEST_FILES CONFIGURE_DEPENDS
    "*_test.cpp"
    "test_utils.cpp"
)

# Add custom command to check for .find() != npos patterns before building tests
add_custom_command(
    OUTPUT ${CMAKE_CURRENT_BINARY_DIR}/find_npos_check.stamp
    COMMAND ${CMAKE_SOURCE_DIR}/scripts/fix_find_npos.sh ${CMAKE_SOURCE_DIR}
    COMMAND ${CMAKE_COMMAND} -E touch ${CMAKE_CURRENT_BINARY_DIR}/find_npos_check.stamp
    DEPENDS ${TEST_FILES}
    COMMENT "Checking for .find() != npos patterns in test files (will auto-fix)"
    VERBATIM
)

# Create a target that depends on the check
add_custom_target(check_find_npos_patterns
    DEPENDS ${CMAKE_CURRENT_BINARY_DIR}/find_npos_check.stamp
)

# Only create tests if BUILD_TESTING is enabled and Catch2 is available
if(BUILD_TESTING AND TARGET Catch2::Catch2WithMain)
    # Create a single test executable for all tests
    add_executable(unit_tests ${TEST_FILES})
    target_link_libraries(unit_tests
        foundation_shell_lib
        Catch2::Catch2WithMain
    )
    
    # Make unit_tests depend on the find/npos pattern check
    # TEMPORARILY DISABLED while fixing tests for bash
    # add_dependencies(unit_tests check_find_npos_patterns)
    
    # Add include directories for test files (using absolute paths)
    target_include_directories(unit_tests PRIVATE
        "${CMAKE_SOURCE_DIR}"
    )
else()
    if(TARGET Catch2::Catch2WithMain)
        set(CATCH2_AVAILABLE "YES")
    else()
        set(CATCH2_AVAILABLE "NO")
    endif()
    message(STATUS "Skipping unit tests - BUILD_TESTING=${BUILD_TESTING}, Catch2 available=${CATCH2_AVAILABLE}")
endif()

include(CTest)

# Only try to discover tests if Catch2 is available and BUILD_TESTING is enabled
if(BUILD_TESTING AND TARGET Catch2::Catch2 AND TARGET unit_tests)
    # Add Catch2 test discovery
    if(catch2_SOURCE_DIR AND EXISTS "${catch2_SOURCE_DIR}/extras/Catch.cmake")
        list(APPEND CMAKE_MODULE_PATH ${catch2_SOURCE_DIR}/extras)
        include(Catch)
        catch_discover_tests(unit_tests)
    else()
        # Fallback: just add the test executable without discovery
        add_test(NAME unit_tests COMMAND unit_tests)
    endif()
endif()
