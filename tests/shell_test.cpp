#include <catch2/catch_all.hpp>
#include "test_utils.hpp"
#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <sstream>
#include <cstdlib>
#include <unistd.h>

// Tests for the basic shell functionality
TEST_CASE("Executes simple command", "[shell]")
{
	std::string output = runShellCommand("echo hello");
	CHECK(output.find("hello") != std::string::npos);
}

TEST_CASE("Executes command with arguments", "[shell]")
{
	std::string output = runShellCommand("echo arg1 arg2 arg3");
	CHECK(output.find("arg1") != std::string::npos);
	CHECK(output.find("arg2") != std::string::npos);
	CHECK(output.find("arg3") != std::string::npos);
}

TEST_CASE("Handles quoted arguments", "[shell]")
{
	std::string output = runShellCommand("echo \"hello world\"");
	CHECK(output.find("hello world") != std::string::npos);
}

TEST_CASE("Handles environment variables", "[shell]")
{
	// Set an environment variable
	setenv("TEST_VAR", "test_value", 1);

	std::string output = runShellCommand("echo $TEST_VAR");
	CHECK(output.find("test_value") != std::string::npos);

	// Clean up
	unsetenv("TEST_VAR");
}

TEST_CASE("Handles home directory", "[shell]")
{
	std::string output = runShellCommand("echo ~");
	CHECK(output.find(getenv("HOME")) != std::string::npos);
}

TEST_CASE("Handles tilde expansion", "[shell]")
{
	std::string output = runShellCommand("echo ~/test");
	std::string expected = std::string(getenv("HOME")) + "/test";
	CHECK(output.find(expected) != std::string::npos);
}

TEST_CASE("Handles escaped characters", "[shell]")
{
	std::string output = runShellCommand("echo hello\\\\world");
	CHECK(output.find("hello\\world") != std::string::npos);
}

TEST_CASE("Handles invalid command", "[shell]")
{
	std::string output = runShellCommand("nonexistentcommand123");
	CHECK(output.find("Command not found") != std::string::npos);
}

// Tests for built-in commands
TEST_CASE("Built-in exit", "[shell][builtins]")
{
	std::string output = runShellCommand("echo before_exit\nexit\necho after_exit");
	CHECK(output.find("before_exit") != std::string::npos);
	CHECK(output.find("after_exit") == std::string::npos);
}

// Test for creating temporary files
TEST_CASE("Creates temporary file", "[shell][redirection]")
{
	// Create a temporary file path
	char tempPath[] = "/tmp/shell_test_XXXXXX";
	int fd = mkstemp(tempPath);
	REQUIRE(fd != -1);
	INFO("Failed to create temporary file");
	close(fd);

	// Write to the file via the shell
	std::string command = "echo 'test content' > " + std::string(tempPath);
	runShellCommand(command);

	// Read back the file contents
	std::ifstream file(tempPath);
	REQUIRE(file.is_open());
	INFO("Failed to open temporary file");

	std::string content;
	std::getline(file, content);
	file.close();

	// The shell should implement redirection correctly
	CHECK(content == "'test content'");
	INFO("Expected file to contain redirected content");

	// Clean up
	unlink(tempPath);
}

// Test for handling combinations of quotes, escapes, and variables
TEST_CASE("Handles complex command line", "[shell]")
{
	setenv("TEST_VAR", "test_value", 1);
	std::string output = runShellCommand("echo \"$TEST_VAR in quotes\" and \\'escaped\\' characters");
	CHECK(output.find("test_value in quotes") != std::string::npos);
	CHECK(output.find("'escaped'") != std::string::npos);
	unsetenv("TEST_VAR");
}

// Test to confirm shell is stateless - variables don't persist between commands
TEST_CASE("Confirm stateless", "[shell]")
{
	/*
	 * This test verifies that our shell is stateless, meaning it doesn't
	 * preserve variables between commands. A stateful shell like bash would
	 * allow variables to be set in one line and used in the next,
	 * but our stateless shell should not.
	 */

	// First, let's confirm environment variables work correctly
	// (these are maintained by the OS, not shell state)
	setenv("STATELESS_TEST_VAR", "test_environment_value", 1);
	std::string envOutput = runShellCommand("echo $STATELESS_TEST_VAR");
	CHECK(envOutput.find("test_environment_value") != std::string::npos);

	// Now for the actual statelessness test:
	// Run two commands: one attempting to set a variable and one trying to echo it
	std::string output = runShellCommand("INTERNAL_VAR=unique_test_string\necho $INTERNAL_VAR");

	// Parse the output line by line
	std::istringstream iss(output);
	std::string line;
	std::string echoOutput;

	// Skip past the prompt and command lines
	while (std::getline(iss, line))
	{
		if (line.find("echo $INTERNAL_VAR") != std::string::npos)
		{
			// Get the next line, which should be the output of echo
			if (std::getline(iss, echoOutput))
			{
				break;
			}
		}
	}

	// The important part: in a stateless shell, executing "echo $INTERNAL_VAR" should
	// not print "unique_test_string" because the INTERNAL_VAR assignment doesn't persist
	CHECK(echoOutput.find("unique_test_string") == std::string::npos);
	INFO("Shell incorrectly preserved variable value between commands");

	// Clean up
	unsetenv("STATELESS_TEST_VAR");
}
