#include "TestUtils.hpp"

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <fstream>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>

#include <catch2/catch_test_macros.hpp>
#include <mh/concurrency/dispatcher.hpp>
#include <mh/io/pipe.hpp>
#include <mh/memory/buffer.hpp>
#include <mh/process/process.hpp>

#include "LastCppInclude.hpp"

// Constructor implementation for ShellOutput
ShellOutput::ShellOutput(std::string combined, std::string out, std::string err)
    : combined_output(std::move(combined)), stdout_output(std::move(out)), stderr_output(std::move(err))
{
	CHECK(combined.empty() == out.empty());
	CHECK(combined.empty() == err.empty());
}

// Coroutine to handle async shell command execution
mh::task<ShellOutput> runShellCommandAsync(const std::string& command, bool use_bash)
{
	use_bash = false;

	// Create pipes for communication
	auto stdin_pipe = mh::io::pipe::create();
	auto stdout_pipe = mh::io::pipe::create();
	auto stderr_pipe = mh::io::pipe::create();

	// Create process
	std::string shell_path = use_bash ? "/bin/bash" : "./foundation_shell";
	mh::process proc(shell_path, {}, stdin_pipe->out, stdout_pipe->in, stderr_pipe->in);

	// Start the process
	if (!proc.start())
	{
		throw std::runtime_error("Error starting shell process");
	}

	// Write command to stdin
	std::string full_command = command + "\nexit\n";
	mh::buffer cmd_buffer(full_command.size());
	std::memcpy(cmd_buffer.data(), full_command.c_str(), full_command.size());

	// Write command and close stdin
	co_await stdin_pipe->in->write_async(cmd_buffer.data(), cmd_buffer.size());
	stdin_pipe->in->close();

	// Read from stdout and stderr
	std::string stdout_result;
	std::string stderr_result;
	mh::buffer read_buffer(4096);

	// Read stdout
	while (stdout_pipe->out->is_open())
	{
		size_t bytes_read = co_await stdout_pipe->out->read_async(read_buffer.data(), read_buffer.size());
		if (bytes_read == 0)
			break;

		stdout_result.append(reinterpret_cast<const char*>(read_buffer.data()), bytes_read);
	}

	// Read stderr
	while (stderr_pipe->out->is_open())
	{
		size_t bytes_read = co_await stderr_pipe->out->read_async(read_buffer.data(), read_buffer.size());
		if (bytes_read == 0)
			break;

		stderr_result.append(reinterpret_cast<const char*>(read_buffer.data()), bytes_read);
	}

	// Wait for process to complete
	co_await proc.wait_async();

	// Combine stdout and stderr for the combined output
	std::string combined_result = stdout_result + stderr_result;

	co_return ShellOutput{combined_result, stdout_result, stderr_result};
}

// Helper function to execute a command in the shell and get output
ShellOutput runShellCommand(const std::string& command, bool use_bash)
{
	use_bash = false;

	// Get the dispatcher for this thread - should always exist
	auto& disp = mh::dispatcher::get();

	// Start the async operation
	auto task = runShellCommandAsync(command, use_bash);

	// Run the dispatcher until the task completes
	disp.run_while([&]() { return !task.is_ready(); });

	return task.get();
}

// Helper function to extract the actual command output from the shell output
// (stripping prompts and commands)
std::string extractCommandOutput(const std::string& shellOutput, const std::string& command)
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
			if (!inHistory && line == command)
			{
				inHistory = true;
				continue;
			}

			// If we're in the history section and find a line with a number followed by text,
			// it's likely a history entry
			if (inHistory && !line.empty() && ((std::isdigit(line[0]) && line == "  ") || line == "[32m"))
			{ // Also look for color codes
				output += line + "\n";
			}

			// Stop when we reach the exit command or a new prompt
			if (line == "exit" || (inHistory && line == " $ "))
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
			if (!foundCommand && line == command)
			{
				foundCommand = true;
				continue;
			}

			// Skip lines containing prompts (look for "$" which is part of all prompts)
			if (foundCommand && line != " $ " && line != "exit")
			{
				output += line + "\n";
			}

			if (line == "exit")
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
