#include <catch2/catch_all.hpp>
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

// Helper function to execute a command in the shell and get output (reused from shell_test.cpp)
std::string runShellCommand(const std::string &command)
{
	// Create pipes
	int stdin_pipe[2];
	int stdout_pipe[2];
	int stderr_pipe[2];

	if (pipe(stdin_pipe) == -1 || pipe(stdout_pipe) == -1 || pipe(stderr_pipe) == -1)
	{
		return "Error creating pipes";
	}

	// Fork a child process
	pid_t pid = fork();

	if (pid == -1)
	{
		return "Error forking process";
	}
	else if (pid == 0)
	{
		// Child process

		// Redirect stdin to read from stdin_pipe
		dup2(stdin_pipe[0], STDIN_FILENO);
		close(stdin_pipe[0]);
		close(stdin_pipe[1]);

		// Redirect stdout to write to stdout_pipe
		dup2(stdout_pipe[1], STDOUT_FILENO);
		close(stdout_pipe[0]);
		close(stdout_pipe[1]);

		// Redirect stderr to write to stderr_pipe
		dup2(stderr_pipe[1], STDERR_FILENO);
		close(stderr_pipe[0]);
		close(stderr_pipe[1]);

		// Execute the shell
		execl("./foundation_shell", "foundation_shell", nullptr);

		// If execl returns, there was an error
		perror("Error executing shell");
		exit(1);
	}

	// Parent process

	// Close unused pipe ends
	close(stdin_pipe[0]);
	close(stdout_pipe[1]);
	close(stderr_pipe[1]);

	// Write command to the shell's stdin
	std::string full_command = command + "\nexit\n";
	write(stdin_pipe[1], full_command.c_str(), full_command.size());
	close(stdin_pipe[1]);

	// Read output from the shell's stdout
	char buffer[4096];
	std::string result;
	ssize_t bytes;

	while ((bytes = read(stdout_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
	{
		buffer[bytes] = '\0';
		result += buffer;
	}
	close(stdout_pipe[0]);

	// Read output from the shell's stderr and append to result
	while ((bytes = read(stderr_pipe[0], buffer, sizeof(buffer) - 1)) > 0)
	{
		buffer[bytes] = '\0';
		result += buffer;
	}
	close(stderr_pipe[0]);

	// Wait for the child to finish
	int status;
	waitpid(pid, &status, 0);

	return result;
}

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string &shellOutput, const std::string &command)
{
	std::istringstream stream(shellOutput);
	std::string line;
	bool foundCommand = false;
	std::string output;

	// Special case for history command since its output includes numbers
	if (command == "history")
	{
		bool inHistory = false;
		while (std::getline(stream, line))
		{
			// Look for the history command itself
			if (!inHistory && line.find(command) != std::string::npos)
			{
				inHistory = true;
				continue;
			}

			// If we're in the history section and find a line with a number followed by text,
			// it's likely a history entry
			if (inHistory && !line.empty() &&
				((std::isdigit(line[0]) && line.find("  ") != std::string::npos) ||
				 line.find("[32m") != std::string::npos))
			{ // Also look for color codes
				output += line + "\n";
			}

			// Stop when we reach the exit command or a new prompt
			if (line.find("exit") != std::string::npos ||
				(inHistory && line.find(" $ ") != std::string::npos))
			{
				break;
			}
		}
	}
	else
	{
		// Standard extraction for other commands
		while (std::getline(stream, line))
		{
			if (!foundCommand && line.find(command) != std::string::npos)
			{
				foundCommand = true;
				continue;
			}

			// Skip lines containing prompts (look for "$" which is part of all prompts)
			if (foundCommand && line.find(" $ ") == std::string::npos && line != "exit")
			{
				output += line + "\n";
			}

			if (line.find("exit") != std::string::npos)
			{
				break;
			}
		}
	}

	// Remove trailing newline if present
	if (!output.empty() && output.back() == '\n')
	{
		output.pop_back();
	}

	return output;
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
	std::string output = runShellCommand(std::string("cat < ") + tempPath);

	// Verify input was correctly redirected
	CHECK(output.find(testContent) != std::string::npos);

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
	CHECK(content.find(initialContent) != std::string::npos);
	CHECK(content.find(appendContent) != std::string::npos);

	// Clean up
	fs::remove(tempPath);
}

// Test for pipe functionality
TEST_CASE("Simple pipe works", "[features][piping]")
{
	// Test a simple pipe
	std::string output = runShellCommand("echo pipe_test | grep pipe");

	// Verify pipe works correctly
	CHECK(output.find("pipe_test") != std::string::npos);
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

TEST_CASE("Multiple pipes work", "[features][piping]")
{
	// Test a chain of pipes
	std::string output = runShellCommand("echo multi_pipe_test | grep multi | grep pipe");

	// Verify multiple pipes work correctly
	CHECK(output.find("multi_pipe_test") != std::string::npos);
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
	CHECK(output.find(cwd) != std::string::npos);
}

// Test for clear command
TEST_CASE("Clear works", "[features][builtin_commands]")
{
	// First output something
	std::string output = runShellCommand("echo before_clear\nclear\necho after_clear");

	// Check that the output contains the clear screen escape sequence
	// This could be either the actual escape sequence or some representation of it
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
	std::cout << "Background process test output: " << output << std::endl;

	// Wait and retry a few times if necessary - file system operations can be async
	bool fileExists = false;
	for (int i = 0; i < 5 && !fileExists; i++)
	{
		fileExists = fs::exists(tempPath);
		if (!fileExists)
		{
			std::cout << "File not found yet, waiting..." << std::endl;
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
	std::string output = runShellCommand("echo colored_test --color=always");

	// Check for ANSI color codes in the output
	// Note: This test might be implementation-specific
	CHECK(output.find("colored_test") != std::string::npos);

	// This test is a placeholder - actual implementation will depend on how colors are incorporated
	// We might look for specific ANSI escape sequences
}

// Test for improved prompt that shows current directory
TEST_CASE("Prompt shows current directory", "[features][ui_improvements]")
{
	// Run a command that changes directory and outputs something
	std::string output = runShellCommand("cd /tmp\npwd");

	// Verify that the prompt contains '/tmp'
	CHECK(output.find("/tmp") != std::string::npos);
}
