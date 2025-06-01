#include <catch2/catch_all.hpp>
#include "test_utils.hpp"
#include "Config.hpp"
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
	CHECK(output == "hello");
}

TEST_CASE("Executes command with arguments", "[shell]")
{
	std::string output = runShellCommand("echo arg1 arg2 arg3");
	CHECK(output == "arg1 arg2 arg3");
}

TEST_CASE("Handles quoted arguments", "[shell]")
{
	std::string output = runShellCommand("echo \"hello world\"");
	CHECK(output == "hello world");
}

TEST_CASE("Handles environment variables", "[shell]")
{
	// Set an environment variable
	setenv("TEST_VAR", "test_value", 1);

	std::string output = runShellCommand("echo $TEST_VAR");
	CHECK(output == "test_value");

	// Clean up
	unsetenv("TEST_VAR");
}

TEST_CASE("Handles home directory", "[shell]")
{
	std::string output = runShellCommand("echo ~");
	CHECK(output == getenv("HOME"));
}

TEST_CASE("Handles tilde expansion", "[shell]")
{
	std::string output = runShellCommand("echo ~/test");
	std::string expected = std::string(getenv("HOME")) + "/test";
	CHECK(output == expected);
}

TEST_CASE("Handles escaped characters", "[shell]")
{
	std::string output = runShellCommand("echo hello\\\\world");
	CHECK(output == "hello\\world");
}

TEST_CASE("Handles invalid command", "[shell]")
{
	std::string output = runShellCommand("nonexistentcommand123");
	CHECK(output == ErrorMessages::COMMAND_NOT_FOUND);
}

// Tests for built-in commands
TEST_CASE("Built-in exit", "[shell][builtins]")
{
	std::string output = runShellCommand("echo before_exit\nexit\necho after_exit");
	CHECK(output == "before_exit");
	CHECK(output != "after_exit");
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
	CHECK(output == "test_value in quotes and 'escaped' characters");
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
	CHECK(envOutput == "test_environment_value");

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
		if (line == "echo $INTERNAL_VAR")
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
	CHECK(echoOutput != "unique_test_string");
	INFO("Shell incorrectly preserved variable value between commands");

	// Clean up
	unsetenv("STATELESS_TEST_VAR");
}

// Test Case: Basic Command Execution - Exit Status Propagation (from test plan)
TEST_CASE("Exit status propagation", "[shell][exitstatus]")
{
	// Test successful command
	std::string output = runShellCommand("true");
	// Note: We can't directly test exit status in this framework, but we can test behavior

	// Test failed command
	output = runShellCommand("false");
	// The command should execute without crashing, even if it returns non-zero

	// Test non-existent command returns 127
	output = runShellCommand("command_that_definitely_does_not_exist_12345");
	CHECK(output == ErrorMessages::COMMAND_NOT_FOUND);
}

// Test Case: Command Not Found scenarios (from test plan)
TEST_CASE("Command not found scenarios", "[shell][errors]")
{
	// Test completely non-existent command
	std::string output = runShellCommand("nonexistent_command");
	CHECK(output == ErrorMessages::COMMAND_NOT_FOUND);

	// Test misspelled command
	output = runShellCommand("ech hello"); // misspelled "echo"
	CHECK(output == ErrorMessages::COMMAND_NOT_FOUND);

	// Test case sensitivity
	output = runShellCommand("Echo hello"); // wrong case
	CHECK(output == ErrorMessages::COMMAND_NOT_FOUND);
}

// Test Case: Built-in pwd command (from test plan)
TEST_CASE("Built-in pwd command", "[shell][builtins]")
{
	std::string output = runShellCommand("pwd");
	// Should contain current working directory
	CHECK(!output.empty());
	CHECK(output == "/"); // Should contain path separator
}

// Test Case: Built-in cd command (from test plan)
TEST_CASE("Built-in cd command", "[shell][builtins]")
{
	// Test cd to home directory (no args)
	std::string output = runShellCommand("cd\npwd");
	std::string homePath = getenv("HOME");
	CHECK(output == homePath);

	// Test cd to /tmp
	output = runShellCommand("cd /tmp\npwd");
	CHECK(output == "/tmp");

	// Test cd to previous directory (-)
	output = runShellCommand("cd /tmp\ncd /\ncd -\npwd");
	CHECK(output == "/tmp");

	// Test cd to non-existent directory
	output = runShellCommand("cd /nonexistent_directory_12345");
	CHECK(output == ErrorMessages::FS_ENTRY_NOT_FOUND);
}

// Test Case: Quote types and escaping (from test plan)
TEST_CASE("Quote types and character escaping", "[shell][quotes]")
{
	// Test single quotes preserve everything literally
	setenv("TEST_VAR", "expanded", 1);
	std::string output = runShellCommand("echo 'single quotes preserve $TEST_VAR'");
	CHECK(output == "$TEST_VAR"); // Should be literal
	CHECK(output != "expanded"); // Should not expand

	// Test double quotes allow variable expansion
	output = runShellCommand("echo \"double quotes allow $TEST_VAR\"");
	CHECK(output == "expanded"); // Should expand

	// Test backslash escaping
	output = runShellCommand("echo hello\\ world");
	CHECK(output == "hello world");

	// Test escaped dollar sign
	output = runShellCommand("echo \\$TEST_VAR");
	CHECK(output == "$TEST_VAR"); // Should be literal
	CHECK(output != "expanded"); // Should not expand

	unsetenv("TEST_VAR");
}

// Test Case: Multiple arguments and field splitting (from test plan)
TEST_CASE("Argument handling and field splitting", "[shell][args]")
{
	// Test multiple space-separated arguments
	std::string output = runShellCommand("echo arg1 arg2 arg3");
	CHECK(output == "arg1 arg2 arg3");

	// Test arguments with extra spaces
	output = runShellCommand("echo   arg1    arg2   arg3   ");
	CHECK(output == "arg1 arg2 arg3");

	// Test quoted arguments preserve spaces
	output = runShellCommand("echo \"multiple   spaces   preserved\"");
	CHECK(output == "multiple   spaces   preserved");
}

// Test Case: Path resolution and executable discovery (from test plan)
TEST_CASE("Path resolution", "[shell][path]")
{
	// Test absolute path execution
	std::string output = runShellCommand("/bin/echo absolute_path_test");
	CHECK(output == "absolute_path_test");

	// Test PATH search (echo should be found in PATH)
	output = runShellCommand("echo path_search_test");
	CHECK(output == "path_search_test");

	// Test current directory execution (assuming echo exists there, which it won't)
	output = runShellCommand("./nonexistent_in_current_dir");
	CHECK(output == ErrorMessages::COMMAND_NOT_FOUND);
}

// Test Case: File existence tests (from test plan)
TEST_CASE("File operations and existence", "[shell][files]")
{
	// Create a temporary file for testing
	char tempPath[] = "/tmp/shell_test_file_XXXXXX";
	int fd = mkstemp(tempPath);
	REQUIRE(fd != -1);
	close(fd);

	// Test file existence with ls
	std::string output = runShellCommand("ls " + std::string(tempPath));
	CHECK(output == tempPath);

	// Test directory operations
	output = runShellCommand("mkdir /tmp/test_shell_dir");
	output = runShellCommand("ls -d /tmp/test_shell_dir");
	CHECK(output == "/tmp/test_shell_dir");

	// Clean up
	unlink(tempPath);
	rmdir("/tmp/test_shell_dir");
}

// Test Case: Basic output redirection (from test plan)
TEST_CASE("Basic output redirection", "[shell][redirection]")
{
	// Create temp file for redirection testing
	char tempPath[] = "/tmp/shell_redir_test_XXXXXX";
	int fd = mkstemp(tempPath);
	REQUIRE(fd != -1);
	close(fd);

	// Test basic output redirection
	std::string command = "echo 'redirected output' > " + std::string(tempPath);
	runShellCommand(command);

	// Read back the file contents
	std::ifstream file(tempPath);
	REQUIRE(file.is_open());
	std::string content;
	std::getline(file, content);
	file.close();

	CHECK(content == "'redirected output'");

	// Test append redirection
	command = "echo 'appended line' >> " + std::string(tempPath);
	runShellCommand(command);

	// Read back again
	std::ifstream file2(tempPath);
	std::string line1, line2;
	std::getline(file2, line1);
	std::getline(file2, line2);
	file2.close();

	CHECK(line1 == "'redirected output'");
	CHECK(line2 == "'appended line'");

	// Clean up
	unlink(tempPath);
}

// Test Case: Special characters in arguments (from test plan)
TEST_CASE("Special characters handling", "[shell][special_chars]")
{
	// Test arguments with special characters in quotes
	std::string output = runShellCommand("echo \"a & b\"");
	CHECK(output == "a & b");

	output = runShellCommand("echo \"a | b\"");
	CHECK(output == "a | b");

	output = runShellCommand("echo \"a ; b\"");
	CHECK(output == "a ; b");

	// Test backslash escaping of special characters
	output = runShellCommand("echo a \\& b");
	CHECK(output == "a & b");
}

// Test Case: Environment variable setting and usage (from grok test plan)
TEST_CASE("Environment variable operations", "[shell][env]")
{
	// Test setting environment variables for single command
	std::string output = runShellCommand("TEST_ENV_VAR=test_value sh -c 'echo $TEST_ENV_VAR'");
	CHECK(output == "test_value");

	// Test that environment variable doesn't persist after command
	output = runShellCommand("sh -c 'echo $TEST_ENV_VAR'");
	CHECK(output == "");

	// Test multiple environment variables
	output = runShellCommand("VAR1=val1 VAR2=val2 sh -c 'echo $VAR1 $VAR2'");
	CHECK(output == "val1 val2");
}

// Test Case: Basic piping (from grok test plan)
TEST_CASE("Basic piping operations", "[shell][pipes]")
{
	// Test simple pipe
	std::string output = runShellCommand("echo 'hello world' | wc -w");
	CHECK(output == "2");

	// Test pipe with grep
	output = runShellCommand("echo -e 'line1\\npattern\\nline3' | grep pattern");
	CHECK(output == "pattern");

	// Test pipe to transform case
	output = runShellCommand("echo 'hello' | tr 'a-z' 'A-Z'");
	CHECK(output == "HELLO");
}

// Test Case: Conditional execution (from grok test plan)
TEST_CASE("Conditional execution", "[shell][conditional]")
{
	// Test && operator - success case
	std::string output = runShellCommand("true && echo 'success'");
	CHECK(output == "success");

	// Test && operator - failure case
	output = runShellCommand("false && echo 'should not print'");
	CHECK(output == "");

	// Test || operator - success case  
	output = runShellCommand("true || echo 'should not print'");
	CHECK(output == "");

	// Test || operator - failure case
	output = runShellCommand("false || echo 'fallback'");
	CHECK(output == "fallback");

	// Test combined && and ||
	output = runShellCommand("true && false || echo 'final'");
	CHECK(output == "final");
}

// Test Case: Input redirection (from gemini aistudio test plan)
TEST_CASE("Input redirection", "[shell][redirection][input]")
{
	// Create temp file with test content
	char tempPath[] = "/tmp/shell_input_test_XXXXXX";
	int fd = mkstemp(tempPath);
	REQUIRE(fd != -1);
	write(fd, "test input line", 15);
	close(fd);

	// Test input redirection
	std::string command = "cat < " + std::string(tempPath);
	std::string output = runShellCommand(command);
	CHECK(output == "test input line");

	// Test word count from input
	command = "wc -w < " + std::string(tempPath);
	output = runShellCommand(command);
	CHECK(output == "3");

	// Clean up
	unlink(tempPath);
}

// Test Case: Multiple pipes (from gemini aistudio test plan)  
TEST_CASE("Multiple pipe operations", "[shell][pipes][complex]")
{
	// Create temp file for pipe testing
	char tempPath[] = "/tmp/shell_pipe_test_XXXXXX";
	int fd = mkstemp(tempPath);
	REQUIRE(fd != -1);
	write(fd, "apple\nbanana\napple\norange", 22);
	close(fd);

	// Test multiple pipes
	std::string command = "cat " + std::string(tempPath) + " | grep apple | wc -l";
	std::string output = runShellCommand(command);
	CHECK(output == "2");

	// Clean up
	unlink(tempPath);
}

// Test Case: Command substitution (from gemini aistudio test plan)
TEST_CASE("Command substitution", "[shell][substitution]")
{
	// Test basic command substitution with $(...)
	std::string output = runShellCommand("echo 'Current dir: $(pwd)'");
	CHECK(output.find("Current dir:") != std::string::npos);

	// Test backtick command substitution
	output = runShellCommand("echo 'Files: `ls | wc -l`'");
	CHECK(output.find("Files:") != std::string::npos);

	// Test command substitution with no output
	output = runShellCommand("echo 'Result: $(true)'");
	CHECK(output == "Result: ");
}

// Test Case: Built-in export command (from gemini aistudio test plan)
TEST_CASE("Built-in export command", "[shell][builtins][export]")
{
	// Test basic export
	std::string output = runShellCommand("export TEST_EXPORT=exported_value\necho $TEST_EXPORT");
	CHECK(output == "exported_value");

	// Test export with spaces
	output = runShellCommand("export TEST_SPACES='value with spaces'\necho \"$TEST_SPACES\"");
	CHECK(output == "value with spaces");

	// Test export display (show all env vars)
	output = runShellCommand("export");
	CHECK(output.find("PATH") != std::string::npos);
}

// Test Case: Built-in unset command (from gemini aistudio test plan)
TEST_CASE("Built-in unset command", "[shell][builtins][unset]")
{
	// Set a variable then unset it
	std::string output = runShellCommand("export TO_UNSET=temporary\necho $TO_UNSET\nunset TO_UNSET\necho $TO_UNSET");
	// The output should contain "temporary" from the first echo, but be empty for the second
	CHECK(output.find("temporary") != std::string::npos);

	// Test unsetting non-existent variable (should not error)
	output = runShellCommand("unset NON_EXISTENT_VAR");
	// Should complete without error
}

// Test Case: Environment variable expansion edge cases (from gemini aistudio test plan)
TEST_CASE("Environment variable expansion", "[shell][env][expansion]")
{
	// Test braces notation
	setenv("PREFIX", "test", 1);
	std::string output = runShellCommand("echo ${PREFIX}_suffix");
	CHECK(output == "test_suffix");

	// Test undefined variable expansion
	output = runShellCommand("echo 'Value: $UNDEFINED_VAR_XYZ'");
	CHECK(output == "Value: ");

	// Test expansion in double quotes
	output = runShellCommand("echo \"PREFIX is: $PREFIX\"");
	CHECK(output == "PREFIX is: test");

	// Test no expansion in single quotes
	output = runShellCommand("echo 'PREFIX is: $PREFIX'");
	CHECK(output == "PREFIX is: $PREFIX");

	unsetenv("PREFIX");
}

// Test Case: Unmatched quotes (from gemini aistudio test plan)
TEST_CASE("Unmatched quotes handling", "[shell][quotes][errors]")
{
	// Test unmatched double quote
	std::string output = runShellCommand("echo \"unclosed quote");
	// Should produce some kind of error or handle gracefully
	CHECK(!output.empty());

	// Test unmatched single quote
	output = runShellCommand("echo 'unclosed quote");
	// Should produce some kind of error or handle gracefully  
	CHECK(!output.empty());
}

// Test Case: Empty arguments (from gemini aistudio test plan)
TEST_CASE("Empty arguments handling", "[shell][args][empty]")
{
	// Test empty string as argument
	std::string output = runShellCommand("echo '' next");
	CHECK(output == "next");

	// Test multiple empty arguments
	output = runShellCommand("echo first '' '' last");
	CHECK(output == "first  last");
}

// Tests for stdout/stderr separation
TEST_CASE("Command output goes to stdout only", "[shell][stdout_stderr]")
{
	ShellOutput output = runShellCommandSeparate("echo hello_world");
	
	// Command output should be in stdout
	CHECK(output.stdout_output == "hello_world\n");
	
	// Prompts and welcome message should be in stderr only
	CHECK(output.stderr_output.find("Welcome to Foundation Shell") != std::string::npos);
	CHECK(output.stderr_output.find("$") != std::string::npos);
	
	// stdout should NOT contain prompts or welcome messages
	CHECK(output.stdout_output.find("Welcome") == std::string::npos);
	CHECK(output.stdout_output.find("$") == std::string::npos);
}

TEST_CASE("Error messages go to stderr only", "[shell][stdout_stderr]")
{
	ShellOutput output = runShellCommandSeparate("nonexistent_command_xyz");
	
	// Error message should be in stderr
	CHECK(output.stderr_output.find("Command not found") != std::string::npos);
	
	// stdout should be empty or only contain whitespace
	bool is_empty_or_whitespace = output.stdout_output.empty() || 
	                              output.stdout_output.find_first_not_of(" \t\n\r") == std::string::npos;
	CHECK(is_empty_or_whitespace);
}

TEST_CASE("Built-in command output goes to stdout", "[shell][stdout_stderr][builtins]")
{
	ShellOutput output = runShellCommandSeparate("pwd");
	
	// pwd output should be in stdout
	CHECK(!output.stdout_output.empty());
	CHECK(output.stdout_output.find("/") != std::string::npos);
	
	// stderr should only contain shell control messages, not the pwd output
	bool stderr_ok = output.stderr_output.find("/") == std::string::npos || 
	                 output.stderr_output.find("Welcome") != std::string::npos;
	CHECK(stderr_ok);
}
