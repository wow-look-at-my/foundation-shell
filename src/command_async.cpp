#include "command.hpp"
#include "CommandChain.hpp"
#include "config.hpp"
#include <iostream>

// External globals from config.hpp
extern ShellConfig shellConfig;
extern bool debugMode;
extern DebugInfo debugInfo;

// Forward declaration for pipeline/pipe handling
static Task<int> executeCommandsWithPipes(const std::vector<Command> &commands);

// Asynchronous version of command chain execution
Task<int> executeCommandChainAsync(CommandChain chain, const ShellConfig &config)
{
	if (chain.commands.empty())
	{
		co_return 0;
	}

	if (debugMode)
	{
		std::cerr << "Debug: Executing command chain with " << chain.commands.size() << " commands" << std::endl;
		debugInfo.pipelineCount++;
	}

	// If there's only one command, execute it directly
	if (chain.commands.size() == 1)
	{
		if (debugMode)
		{
			std::cerr << "Debug: Executing single command: " << chain.commands[0].args[0] << std::endl;

			// Count redirections
			if (!chain.commands[0].inputFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Input redirection from " << chain.commands[0].inputFile << std::endl;
			}
			if (!chain.commands[0].outputFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Output redirection to " << chain.commands[0].outputFile
						  << (chain.commands[0].appendOutput ? " (append)" : "") << std::endl;
			}
			if (!chain.commands[0].errorFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Error redirection to " << chain.commands[0].errorFile
						  << (chain.commands[0].appendError ? " (append)" : "") << std::endl;
			}

			// Count background processes
			if (chain.commands[0].backgroundProcess)
			{
				debugInfo.backgroundProcessCount++;
				std::cerr << "Debug: Running as background process" << std::endl;
			}
		}

		debugInfo.commandCount++;
		bool result = co_await chain.commands[0].execute();
		co_return result ? 0 : 1; // Convert bool to exit status
	}

	// For multiple commands, handle the command chain
	int lastExitStatus = 0;

	for (size_t i = 0; i < chain.commands.size(); ++i)
	{
		// For pipes, we need to set up the pipe
		if (i < chain.operators.size() && chain.operators[i] == ChainOperator::Pipe)
		{
			// Find the end of this pipeline
			size_t pipelineEnd = i;
			while (pipelineEnd < chain.operators.size() && chain.operators[pipelineEnd] == ChainOperator::Pipe)
			{
				pipelineEnd++;
			}

			// Extract the pipeline commands
			std::vector<Command> pipelineCommands;
			for (size_t j = i; j <= pipelineEnd; ++j)
			{
				pipelineCommands.push_back(chain.commands[j]);
			}

			// Execute the pipeline asynchronously (directly handle piped commands)
			lastExitStatus = co_await executeCommandsWithPipes(pipelineCommands);

			// Skip to after this pipeline
			i = pipelineEnd;

			// Check if we need to stop based on the previous operator
			if (i > 0 && i - 1 < chain.operators.size())
			{
				if (chain.operators[i - 1] == ChainOperator::And && lastExitStatus != 0)
				{
					// Stop if previous command failed for AND
					break;
				}
				else if (chain.operators[i - 1] == ChainOperator::Or && lastExitStatus == 0)
				{
					// Stop if previous command succeeded for OR
					break;
				}
			}
		}
		else
		{
			// For non-pipe operators or the last command, execute directly
			debugInfo.commandCount++;

			// Check if we should execute this command based on the previous exit status
			if (i > 0 && i - 1 < chain.operators.size())
			{
				if (chain.operators[i - 1] == ChainOperator::And && lastExitStatus != 0)
				{
					// Skip this command if the previous one failed for AND
					if (debugMode)
					{
						std::cerr << "Debug: Skipping command due to AND operator and previous command failure" << std::endl;
					}
					continue;
				}
				else if (chain.operators[i - 1] == ChainOperator::Or && lastExitStatus == 0)
				{
					// Skip this command if the previous one succeeded for OR
					if (debugMode)
					{
						std::cerr << "Debug: Skipping command due to OR operator and previous command success" << std::endl;
					}
					continue;
				}
			}

			// Execute the command directly and asynchronously
			bool result = co_await chain.commands[i].execute();
			lastExitStatus = result ? 0 : 1; // Convert bool to exit status
		}
	}

	co_return lastExitStatus;
}

// Helper function to execute commands connected by pipes
static Task<int> executeCommandsWithPipes(const std::vector<Command> &commands)
{
	if (commands.empty())
	{
		co_return 0;
	}

	if (debugMode)
	{
		std::cerr << "Debug: Executing piped commands with " << commands.size() << " commands" << std::endl;
		debugInfo.pipelineCount++;
	}

	// If there's only one command, execute it directly
	if (commands.size() == 1)
	{
		debugInfo.commandCount++;
		bool result = co_await commands[0].execute();
		co_return result ? 0 : 1; // Convert bool to exit status
	}

	// For multiple commands, we need to set up pipes
	int lastExitStatus = 0;
	int pipeFds[2];
	int inputFd = STDIN_FILENO;

	for (size_t i = 0; i < commands.size(); ++i)
	{
		// For all but the last command, create a pipe for the output
		if (i < commands.size() - 1)
		{
			if (pipe(pipeFds) == -1)
			{
				std::cerr << "Failed to create pipe\n";
				co_return 1;
			}
		}

		// For all but the last command, output goes to a pipe
		int outputFd = (i < commands.size() - 1) ? pipeFds[1] : STDOUT_FILENO;

		// Execute the current command asynchronously
		bool result = co_await commands[i].execute(inputFd, outputFd);
		lastExitStatus = result ? 0 : 1; // Convert bool to exit status

		// Close the write end of the pipe if we created one
		if (i < commands.size() - 1)
		{
			close(pipeFds[1]);
		}

		// If this isn't the first command, close the previous input
		if (inputFd != STDIN_FILENO)
		{
			close(inputFd);
		}

		// For all but the last command, the next command reads from the pipe
		if (i < commands.size() - 1)
		{
			inputFd = pipeFds[0];
		}
	}

	co_return lastExitStatus;
}
