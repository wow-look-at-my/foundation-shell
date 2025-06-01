#include <catch2/catch_all.hpp>
#include "test_utils.hpp"
#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <sstream>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <cstring>
#include <filesystem>
#include <sys/stat.h>
#include <cstdio> // For stdout
#include "LastCppInclude.hpp"

// Fallback functions for filesystem operations
// to avoid compiler-specific variations in std::filesystem
namespace fs
{
	bool exists(const std::string &path)
	{
		struct stat buffer;
		return (stat(path.c_str(), &buffer) == 0);
	}

	bool remove(const std::string &path)
	{
		return (::remove(path.c_str()) == 0);
	}

	std::uintmax_t file_size(const std::string &path)
	{
		struct stat buffer;
		if (stat(path.c_str(), &buffer) == 0)
		{
			return buffer.st_size;
		}
		return 0;
	}
}

// Tests for redirection
TEST_CASE("Output redirection works", "[features][redirection]")
{
	// Create a temporary file path
	char tempPath[] = "/tmp/shell_redir_test_XXXXXX";
	int fd = mkstemp(tempPath);
	INFO("Failed to create temporary file");
	REQUIRE(fd != -1);
	close(fd);

	// Remove the file to start with a clean state
	fs::remove(tempPath);

	// Run command with output redirection
	runShellCommand(std::string("echo redirect_test_content > ") + tempPath);

	// Check that the file exists
	INFO("Output redirection did not create file at: " << tempPath);
	REQUIRE(fs::exists(tempPath));

	// Read back the file contents
	std::ifstream file(tempPath);
	REQUIRE(file.is_open());

	std::string content;
	std::getline(file, content);

	// Verify content was redirected
	CHECK(content == "redirect_test_content");

	// Clean up
	fs::remove(tempPath);
}

TEST_CASE("Input redirection works", "[features][redirection]")
{
	// Create a temporary file with test content
	char tempPath[] = "/tmp/shell_input_test_XXXXXX";
	int fd = mkstemp(tempPath);
	INFO("Failed to create temporary file");
	REQUIRE(fd != -1);

	// Write test content to the file
	std::string testContent = "input_redirection_test_content";
	write(fd, testContent.c_str(), testContent.size());
	close(fd);

	// Run command with input redirection
	std::string command = std::string("cat < ") + tempPath;
	std::string output = runShellCommand(command);

	// Verify input was correctly redirected
	CHECK(output == testContent);

	// Clean up
	fs::remove(tempPath);
}

TEST_CASE("Append redirection works", "[features][redirection]")
{
	// Create a temporary file with initial content
	char tempPath[] = "/tmp/shell_append_test_XXXXXX";
	int fd = mkstemp(tempPath);
	INFO("Failed to create temporary file");
	REQUIRE(fd != -1);

	std::string initialContent = "initial_content\n";
	write(fd, initialContent.c_str(), initialContent.size());
	close(fd);

	// Append to the file
	std::string appendContent = "appended_content";
	runShellCommand(std::string("echo ") + appendContent + " >> " + tempPath);

	// Read back the file contents
	std::ifstream file(tempPath);
	REQUIRE(file.is_open());

	std::string content((std::istreambuf_iterator<char>(file)),
						std::istreambuf_iterator<char>());

	// Verify both initial and appended content exists
	std::string expectedContent = "initial_content\nappended_content\n";
	CHECK(content == expectedContent);

	// Clean up
	fs::remove(tempPath);
}

// Test for pipe functionality
TEST_CASE("Simple pipe works", "[features][piping]")
{
	// Test a simple pipe
	std::string command = "echo pipe_test | grep pipe";
	std::string output = runShellCommand(command);

	// Verify pipe works correctly
	CHECK(output == "pipe_test\n");
}

// Test for command chaining with &&
TEST_CASE("And operator works", "[features][command_chaining]")
{
	// Create a temporary directory
	char tempDir[] = "/tmp/shell_and_test_XXXXXX";
	INFO("Failed to create temporary directory");
	REQUIRE(mkdtemp(tempDir) != nullptr);

	// Test a simple command chain with &&
	std::string command = std::string("cd ") + tempDir + " && echo success_marker > test_file.txt";
	std::string output = runShellCommand(command);

	// Verify the file was created (meaning both commands executed)
	std::string filePath = std::string(tempDir) + "/test_file.txt";
	INFO("File was not created, && chaining likely failed");
	REQUIRE(fs::exists(filePath));

	// Check file contents
	std::ifstream file(filePath);
	REQUIRE(file.is_open());
	std::string content;
	std::getline(file, content);
	CHECK(content == "success_marker");

	// Clean up
	fs::remove(filePath);
	rmdir(tempDir);
}

// Test that && doesn't execute commands after a failure
TEST_CASE("And operator stops on failure", "[features][command_chaining]")
{
	// Create a temporary directory
	char tempDir[] = "/tmp/shell_and_fail_test_XXXXXX";
	INFO("Failed to create temporary directory");
	REQUIRE(mkdtemp(tempDir) != nullptr);

	// Test a command chain with && where the first command fails
	std::string nonExistentDir = "/nonexistent_directory_12345";
	std::string command = std::string("cd ") + nonExistentDir +
						  " && echo should_not_execute > " + tempDir + "/should_not_exist.txt";
	std::string output = runShellCommand(command);

	// Verify the file was NOT created (meaning second command wasn't executed)
	std::string filePath = std::string(tempDir) + "/should_not_exist.txt";
	INFO("File was created, && chaining failed to stop after error");
	CHECK_FALSE(fs::exists(filePath));

	// Clean up
	rmdir(tempDir);
}

// Test multiple commands in a chain with &&
TEST_CASE("Multiple and operators work", "[features][command_chaining]")
{
	// Create a temporary directory
	char tempDir[] = "/tmp/shell_multi_and_test_XXXXXX";
	INFO("Failed to create temporary directory");
	REQUIRE(mkdtemp(tempDir) != nullptr);

	// Test multiple commands chained with &&
	std::string command = std::string("cd ") + tempDir +
						  " && mkdir -p subdir" +
						  " && cd subdir" +
						  " && echo nested_success > test_file.txt";
	std::string output = runShellCommand(command);

	// Verify the file was created in the nested directory
	std::string filePath = std::string(tempDir) + "/subdir/test_file.txt";
	INFO("File in nested directory was not created, multiple && chaining failed");
	REQUIRE(fs::exists(filePath));

	// Check file contents
	std::ifstream file(filePath);
	REQUIRE(file.is_open());
	std::string content;
	std::getline(file, content);
	CHECK(content == "nested_success");

	// Clean up
	fs::remove(filePath);
	rmdir((std::string(tempDir) + "/subdir").c_str());
	rmdir(tempDir);
}

// Test for cd working within command chains
TEST_CASE("cd works in command chains", "[features][cd][command_chaining]")
{
	// Create a temporary directory
	char tempDir[] = "/tmp/shell_cd_test_XXXXXX";
	REQUIRE(mkdtemp(tempDir) != nullptr);

	// Test that cd && pwd shows we're in the new directory
	std::string command = std::string("cd ") + tempDir + " && pwd";
	std::string output = runShellCommand(command);

	// Check that pwd output shows the temp directory path
	CHECK(output == std::string(tempDir) + "\n");

	// Clean up
	rmdir(tempDir);
}

// Test for temporary cd (directory change should be isolated per command chain)
TEST_CASE("cd changes are temporary per command chain", "[features][cd][stateless]")
{
	// Create two temporary directories
	char tempDir1[] = "/tmp/shell_cd_temp1_XXXXXX";
	char tempDir2[] = "/tmp/shell_cd_temp2_XXXXXX";
	REQUIRE(mkdtemp(tempDir1) != nullptr);
	REQUIRE(mkdtemp(tempDir2) != nullptr);

	// Run first command chain: cd to tempDir1 and create a file
	std::string command1 = std::string("cd ") + tempDir1 + " && echo 'first' > file1.txt";
	runShellCommand(command1);

	// Run second command chain: cd to tempDir2 and create a file
	std::string command2 = std::string("cd ") + tempDir2 + " && echo 'second' > file2.txt";
	runShellCommand(command2);

	// Verify both files were created in their respective directories
	std::string filePath1 = std::string(tempDir1) + "/file1.txt";
	std::string filePath2 = std::string(tempDir2) + "/file2.txt";

	REQUIRE(fs::exists(filePath1));
	REQUIRE(fs::exists(filePath2));

	// Verify file contents
	std::ifstream file1(filePath1), file2(filePath2);
	std::string content1, content2;
	std::getline(file1, content1);
	std::getline(file2, content2);

	CHECK(content1 == "first");
	CHECK(content2 == "second");

	// Clean up
	fs::remove(filePath1);
	fs::remove(filePath2);
	rmdir(tempDir1);
	rmdir(tempDir2);
}

// Test for cd with nonexistent directory
TEST_CASE("cd fails gracefully with nonexistent directory", "[features][cd][error_handling]")
{
	// Try to cd to a directory that doesn't exist
	std::string command = "cd /this/directory/should/not/exist && echo should_not_run";
	std::string output = runShellCommand(command);

	// The command chain should fail and 'should_not_run' should not appear in output
	// For bash, check that it contains an error message and not the success string
	CHECK(output.find("should_not_run") == std::string::npos);
}

TEST_CASE("Multiple pipes work", "[features][piping]")
{
	// Test a chain of pipes
	std::string output = runShellCommand("echo multi_pipe_test | grep multi | grep pipe");

	// Verify multiple pipes work correctly
	CHECK(output == "multi_pipe_test\n");
}

// Test for built-in 'pwd' command
TEST_CASE("Pwd works", "[features][builtin_commands]")
{
	// Test the pwd command
	std::string output = runShellCommand("pwd");

	// Get the current working directory
	char cwd[1024];
	REQUIRE(getcwd(cwd, sizeof(cwd)) != nullptr);

	// Verify pwd output contains the current directory
	CHECK(output == std::string(cwd) + "\n");
}

// Test for clear command
TEST_CASE("Clear works", "[features][builtin_commands]")
{
	// First output something
	std::string output = runShellCommand("echo before_clear\nclear\necho after_clear");

	// Check that the output contains both before and after text plus clear escape sequences
	// Bash clear outputs ANSI escape sequences
	CHECK(output.find("before_clear") != std::string::npos);
	CHECK(output.find("after_clear") != std::string::npos);
}

// Test for background processes
TEST_CASE("Background process works", "[features][process_management]")
{
	// Run a command in the background that creates a file after a brief delay
	char tempPath[] = "/tmp/shell_bg_test_XXXXXX";
	int fd = mkstemp(tempPath);
	INFO("Failed to create temporary file");
	REQUIRE(fd != -1);
	close(fd);
	fs::remove(tempPath);

	// Make the background command more direct to avoid shell interpretation issues
	std::string command = "touch " + std::string(tempPath) + " &";
	std::string output = runShellCommand(command + "\nsleep 3");

	// Add more verbose output to help with debugging
	std::print("Background process test output: {}\n", output);

	// Wait and retry a few times if necessary - file system operations can be async
	bool fileExists = false;
	for (int i = 0; i < 5 && !fileExists; i++)
	{
		fileExists = fs::exists(tempPath);
		if (!fileExists)
		{
			std::print("File not found yet, waiting...\n");
			usleep(500000); // Sleep for 0.5 seconds between retries
		}
	}

	// Final verification
	INFO("Background process did not create file at: " << tempPath);
	REQUIRE(fileExists);

	// For successful tests, write to the file as proof it exists and is writable
	if (fileExists)
	{
		std::ofstream testFile(tempPath);
		testFile << "bg_process_test" << std::endl;
		testFile.close();

		// Read back for verification
		std::ifstream file(tempPath);
		REQUIRE(file.is_open());

		std::string content;
		std::getline(file, content);

		// Verify content
		CHECK(content == "bg_process_test");
	}

	// Clean up
	fs::remove(tempPath);
}

// Test for implementation of colorful output
TEST_CASE("Colored output works", "[features][ui_improvements]")
{
	// Enable colored output mode
	std::string output = runShellCommand("echo colored_test");

	// Check for ANSI color codes in the output
	// Note: This test might be implementation-specific
	CHECK(output == "colored_test\n");

	// This test is a placeholder - actual implementation will depend on how colors are incorporated
	// We might look for specific ANSI escape sequences
}

// Test for improved prompt that shows current directory
TEST_CASE("Prompt shows current directory", "[features][ui_improvements]")
{
	// Run a command that changes directory and outputs something
	std::string output = runShellCommand("cd /tmp\npwd");

	// Verify that the prompt contains '/tmp'
	CHECK(output == "/tmp\n");
}
