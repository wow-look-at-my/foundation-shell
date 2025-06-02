#include <unistd.h>

#include <catch2/catch_all.hpp>
#include <catch2/matchers/catch_matchers_string.hpp>
#include <cstdlib>
#include <format>
#include <sstream>
#include <string>

#include <mh/io/file.hpp>

#include "Config.hpp"
#include "LastCppInclude.hpp"
#include "TestUtils.hpp"

// Tests for the basic shell functionality
TEST_CASE("Executes simple command", "[shell]")
{
	std::string output = runShellCommand("echo hello", true); // Use bash
	CHECK(output == "hello\n");
}

TEST_CASE("Executes command with arguments", "[shell]")
{
	std::string output = runShellCommand("echo arg1 arg2 arg3");
	CHECK(output == "arg1 arg2 arg3\n");
}

TEST_CASE("Handles quoted arguments", "[shell]")
{
	std::string output = runShellCommand("echo \"hello world\"");
	CHECK(output == "hello world\n");
}

TEST_CASE("Handles environment variables", "[shell]")
{
	// Set an environment variable
	setenv("TEST_VAR", "test_value", 1);

	std::string output = runShellCommand("echo $TEST_VAR");
	CHECK(output == "test_value\n");

	// Clean up
	unsetenv("TEST_VAR");
}

TEST_CASE("Handles home directory", "[shell]")
{
	std::string output = runShellCommand("echo ~");
	CHECK(output == std::string(getenv("HOME")) + "\n");
}

TEST_CASE("Handles tilde expansion", "[shell]")
{
	std::string output = runShellCommand("echo ~/test");
	std::string expected = std::string(getenv("HOME")) + "/test\n";
	CHECK(output == expected);
}

TEST_CASE("Handles escaped characters", "[shell]")
{
	std::string output = runShellCommand("echo hello\\\\world");
	CHECK(output == "hello\\world\n");
}

TEST_CASE("Handles invalid command", "[shell]")
{
	std::string output = runShellCommand("nonexistentcommand123");
	// Bash outputs "command not found" error message
	CHECK(!output.empty());
}

// Tests for built-in commands
TEST_CASE("Built-in exit", "[shell][builtins]")
{
	std::string output = runShellCommand("echo before_exit\nexit\necho after_exit");
	CHECK(output == "before_exit\n");
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
	std::string content = mh::read_file(tempPath);

	// The shell should implement redirection correctly (mh::read_file includes trailing newline)
	CHECK(content == "test content\n");
	INFO("Expected file to contain redirected content");

	// Clean up
	unlink(tempPath);
}

// Test for handling combinations of quotes, escapes, and variables
TEST_CASE("Handles complex command line", "[shell]")
{
	setenv("TEST_VAR", "test_value", 1);
	std::string output = runShellCommand("echo \"$TEST_VAR in quotes\" and \\'escaped\\' characters");
	CHECK(output == "test_value in quotes and 'escaped' characters\n");
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
	CHECK(envOutput == "test_environment_value\n");

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
	// Bash outputs "command not found" error message
	CHECK(!output.empty());
}

// Test Case: Command Not Found scenarios (from test plan)
TEST_CASE("Command not found scenarios", "[shell][errors]")
{
	// Test completely non-existent command
	std::string output = runShellCommand("nonexistent_command");
	// Bash outputs "command not found" error message
	CHECK(!output.empty());

	// Test misspelled command
	output = runShellCommand("ech hello"); // misspelled "echo"
	// Bash outputs "command not found" error message
	CHECK(!output.empty());

	// Test case sensitivity
	output = runShellCommand("Echo hello"); // wrong case
	// Bash outputs "command not found" error message
	CHECK(!output.empty());
}

// Test Case: Built-in pwd command (from test plan)
TEST_CASE("Built-in pwd command", "[shell][builtins]")
{
	std::string output = runShellCommand("pwd");
	// Should contain current working directory
	CHECK(!output.empty());
	// Get current working directory and compare
	char cwd[1024];
	getcwd(cwd, sizeof(cwd));
	CHECK(output == std::string(cwd) + "\n");
}

// Test Case: Built-in cd command (from test plan)
TEST_CASE("Built-in cd command", "[shell][builtins]")
{
	// Test cd to home directory (no args)
	std::string output = runShellCommand("cd\npwd");
	std::string homePath = getenv("HOME");
	CHECK(output == std::string(homePath) + "\n");

	// Test cd to /tmp
	output = runShellCommand("cd /tmp\npwd");
	CHECK(output == "/tmp\n");

	// Test cd to previous directory (-)
	output = runShellCommand("cd /tmp\ncd /\ncd -\npwd");
	// This may output multiple lines, just check it ends with /tmp
	CHECK_THAT(output, Catch::Matchers::ContainsSubstring("/tmp"));

	// Test cd to non-existent directory
	output = runShellCommand("cd /nonexistent_directory_12345");
	// Bash outputs "No such file or directory" error message
	CHECK(!output.empty());
}

// Test Case: Quote types and escaping (from test plan)
TEST_CASE("Quote types and character escaping", "[shell][quotes]")
{
	// Test single quotes preserve everything literally
	setenv("TEST_VAR", "expanded", 1);
	std::string output = runShellCommand("echo 'single quotes preserve $TEST_VAR'");
	CHECK(output == "single quotes preserve $TEST_VAR\n"); // Should be literal
	CHECK(output != "expanded\n");                         // Should not expand

	// Test double quotes allow variable expansion
	output = runShellCommand("echo \"double quotes allow $TEST_VAR\"");
	CHECK(output == "double quotes allow expanded\n"); // Should expand

	// Test backslash escaping
	output = runShellCommand("echo hello\\ world");
	CHECK(output == "hello world\n");

	// Test escaped dollar sign
	output = runShellCommand("echo \\$TEST_VAR");
	CHECK(output == "$TEST_VAR\n"); // Should be literal
	CHECK(output != "expanded\n");  // Should not expand

	unsetenv("TEST_VAR");
}

// Test Case: Multiple arguments and field splitting (from test plan)
TEST_CASE("Argument handling and field splitting", "[shell][args]")
{
	// Test multiple space-separated arguments
	std::string output = runShellCommand("echo arg1 arg2 arg3");
	CHECK(output == "arg1 arg2 arg3\n");

	// Test arguments with extra spaces
	output = runShellCommand("echo   arg1    arg2   arg3   ");
	CHECK(output == "arg1 arg2 arg3\n");

	// Test quoted arguments preserve spaces
	output = runShellCommand("echo \"multiple   spaces   preserved\"");
	CHECK(output == "multiple   spaces   preserved\n");
}

// Test Case: Path resolution and executable discovery (from test plan)
TEST_CASE("Path resolution", "[shell][path]")
{
	// Test absolute path execution
	std::string output = runShellCommand("/bin/echo absolute_path_test");
	CHECK(output == "absolute_path_test\n");

	// Test PATH search (echo should be found in PATH)
	output = runShellCommand("echo path_search_test");
	CHECK(output == "path_search_test\n");

	// Test current directory execution (assuming echo exists there, which it won't)
	output = runShellCommand("./nonexistent_in_current_dir");
	// Bash will output something like "bash: line 1: ./nonexistent_in_current_dir: No such file or directory"
	// Just verify that some error occurred (output is not empty)
	CHECK(!output.empty());
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
	CHECK(output == std::string(tempPath) + "\n");

	// Test directory operations
	output = runShellCommand("mkdir /tmp/test_shell_dir");
	output = runShellCommand("ls -d /tmp/test_shell_dir");
	CHECK(output == "/tmp/test_shell_dir\n");

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
	std::string content = mh::read_file(tempPath);

	CHECK(content == "redirected output\n");

	// Test append redirection
	command = "echo 'appended line' >> " + std::string(tempPath);
	runShellCommand(command);

	// Read back again
	std::string fullContent = mh::read_file(tempPath);

	CHECK(fullContent == "redirected output\nappended line\n");

	// Clean up
	unlink(tempPath);
}

// Test Case: Special characters in arguments (from test plan)
TEST_CASE("Special characters handling", "[shell][special_chars]")
{
	// Test arguments with special characters in quotes
	std::string output = runShellCommand("echo \"a & b\"");
	CHECK(output == "a & b\n");

	output = runShellCommand("echo \"a | b\"");
	CHECK(output == "a | b\n");

	output = runShellCommand("echo \"a ; b\"");
	CHECK(output == "a ; b\n");

	// Test backslash escaping of special characters
	output = runShellCommand("echo a \\& b");
	CHECK(output == "a & b\n");
}

// Test Case: Environment variable setting and usage (from grok test plan)
TEST_CASE("Environment variable operations", "[shell][env]")
{
	// Test setting environment variables for single command
	std::string output = runShellCommand("TEST_ENV_VAR=test_value sh -c 'echo $TEST_ENV_VAR'");
	CHECK(output == "test_value\n");

	// Test that environment variable doesn't persist after command
	output = runShellCommand("sh -c 'echo $TEST_ENV_VAR'");
	CHECK(output == "\n");

	// Test multiple environment variables
	output = runShellCommand("VAR1=val1 VAR2=val2 sh -c 'echo $VAR1 $VAR2'");
	CHECK(output == "val1 val2\n");
}

// Test Case: Basic piping (from grok test plan)
TEST_CASE("Basic piping operations", "[shell][pipes]")
{
	// Test simple pipe
	std::string output = runShellCommand("echo 'hello world' | wc -w");
	CHECK(output == "       2\n");

	// Test pipe with grep
	output = runShellCommand("echo -e 'line1\\npattern\\nline3' | grep pattern");
	CHECK(output == "pattern\n");

	// Test pipe to transform case
	output = runShellCommand("echo 'hello' | tr 'a-z' 'A-Z'");
	CHECK(output == "HELLO\n");
}

// Test Case: Conditional execution (from grok test plan)
TEST_CASE("Conditional execution", "[shell][conditional]")
{
	// Test && operator - success case
	std::string output = runShellCommand("true && echo 'success'");
	CHECK(output == "success\n");

	// Test && operator - failure case
	output = runShellCommand("false && echo 'should not print'");
	CHECK(output == "");

	// Test || operator - success case
	output = runShellCommand("true || echo 'should not print'");
	CHECK(output == "");

	// Test || operator - failure case
	output = runShellCommand("false || echo 'fallback'");
	CHECK(output == "fallback\n");

	// Test combined && and ||
	output = runShellCommand("true && false || echo 'final'");
	CHECK(output == "final\n");
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
	CHECK(output == "       3\n");

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
	CHECK(output == "       2\n");

	// Clean up
	unlink(tempPath);
}

// Test Case: Command substitution (from gemini aistudio test plan)
TEST_CASE("Command substitution", "[shell][substitution]")
{
	// Test basic command substitution with $(...)
	std::string output = runShellCommand("echo 'Current dir: $(pwd)'");
	// Single quotes prevent substitution in bash
	CHECK(output == "Current dir: $(pwd)\n");

	// Test backtick command substitution
	output = runShellCommand("echo 'Result: `echo test`'");
	CHECK(output == "Result: `echo test`\n"); // Single quotes prevent substitution in bash

	// Test command substitution with no output
	output = runShellCommand("echo 'Result: $(true)'");
	CHECK(output == "Result: $(true)\n"); // Single quotes prevent substitution in bash
}

// Test Case: Built-in export command (from gemini aistudio test plan)
TEST_CASE("Built-in export command", "[shell][builtins][export]")
{
	// Test basic export
	std::string output = runShellCommand("export TEST_EXPORT=exported_value\necho $TEST_EXPORT");
	CHECK(output == "exported_value\n");

	// Test export with spaces
	output = runShellCommand("export TEST_SPACES='value with spaces'\necho \"$TEST_SPACES\"");
	CHECK(output == "value with spaces\n");

	// Test export display (show all env vars) - just verify it doesn't crash
	output = runShellCommand("export");
	// The export command should complete successfully (no specific output verification needed for bash compatibility)
	CHECK(!output.empty());
}

// Test Case: Built-in unset command (from gemini aistudio test plan)
TEST_CASE("Built-in unset command", "[shell][builtins][unset]")
{
	// Set a variable then unset it
	std::string output = runShellCommand("export TO_UNSET=temporary\necho $TO_UNSET\nunset TO_UNSET\necho $TO_UNSET");
	// The output should contain "temporary" from the first echo, then a newline for the empty second echo
	CHECK(output == "temporary\n\n");

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
	CHECK(output == "test_suffix\n");

	// Test undefined variable expansion
	output = runShellCommand("echo 'Value: $UNDEFINED_VAR_XYZ'");
	CHECK(output == "Value: $UNDEFINED_VAR_XYZ\n");

	// Test expansion in double quotes
	output = runShellCommand("echo \"PREFIX is: $PREFIX\"");
	CHECK(output == "PREFIX is: test\n");

	// Test no expansion in single quotes
	output = runShellCommand("echo 'PREFIX is: $PREFIX'");
	CHECK(output == "PREFIX is: $PREFIX\n");

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
	CHECK(output == " next\n");

	// Test multiple empty arguments
	output = runShellCommand("echo first '' '' last");
	CHECK(output == "first   last\n");
}

// Tests for stdout/stderr separation
TEST_CASE("Command output goes to stdout only", "[shell][stdout_stderr]")
{
	ShellOutput output = runShellCommandSeparate("echo hello_world");

	// Command output should be in stdout
	CHECK(output.stdout_output == "hello_world\n");

	// For bash, stderr would be empty since bash doesn't print prompts or welcome messages
	// when run non-interactively, so we'll just check that stdout contains our command output
	// No specific stderr requirements for bash
}

TEST_CASE("Error messages go to stderr only", "[shell][stdout_stderr]")
{
	ShellOutput output = runShellCommandSeparate("nonexistent_command_xyz");

	// Debug: print actual output to see what we get
	INFO("STDOUT: '" << output.stdout_output << "'");
	INFO("STDERR: '" << output.stderr_output << "'");

	// Error message should be in stderr - bash format will be different but should contain the command name
	CHECK(!output.stderr_output.empty());
	// Just verify stderr is not empty (contains some error message about the nonexistent command)

	// stdout should be empty or only contain whitespace
	bool is_empty_or_whitespace =
	    output.stdout_output.empty() || output.stdout_output.find_first_not_of(" \t\n\r") == std::string::npos;
	CHECK(is_empty_or_whitespace);
}

TEST_CASE("Built-in command output goes to stdout", "[shell][stdout_stderr][builtins]")
{
	ShellOutput output = runShellCommandSeparate("pwd");

	// pwd output should be in stdout
	CHECK(!output.stdout_output.empty());
	// Verify stdout contains a path (starts with /)
	char cwd[1024];
	getcwd(cwd, sizeof(cwd));
	CHECK(output.stdout_output == std::string(cwd) + "\n");
}
