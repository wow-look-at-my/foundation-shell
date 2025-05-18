#include "Command.hpp"
#include "CommandChain.hpp"
#include "Config.hpp"
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
	if (chain.empty())
	{
		co_return 0;
	}

	if (debugMode)
	{
		std::cerr << "Debug: Executing command chain with " << chain.size() << " commands" << std::endl;
		debugInfo.pipelineCount++;
	}

	// If there's only one command, execute it directly
	if (chain.size() == 1)
	{
		if (debugMode)
		{
			std::cerr << "Debug: Executing single command: " << chain.getCommand(0).args[0] << std::endl;

			// Count redirections
			if (!chain.getCommand(0).inputFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Input redirection from " << chain.getCommand(0).inputFile << std::endl;
			}
			if (!chain.getCommand(0).outputFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Output redirection to " << chain.getCommand(0).outputFile
						  << (chain.getCommand(0).appendOutput ? " (append)" : "") << std::endl;
			}
			if (!chain.getCommand(0).errorFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Error redirection to " << chain.getCommand(0).errorFile
						  << (chain.getCommand(0).appendError ? " (append)" : "") << std::endl;
			}

			// Count background processes
			if (chain.getCommand(0).backgroundProcess)
			{
				debugInfo.backgroundProcessCount++;
				std::cerr << "Debug: Running as background process" << std::endl;
			}
		}

		debugInfo.commandCount++;
		bool result = co_await chain.getCommand(0).executeAsync();
		co_return result ? 0 : 1; // Convert bool to exit status
	}

	// For multiple commands, handle the command chain
	int lastExitStatus = 0;

	for (size_t i = 0; i < chain.size(); ++i)
	{
		// For pipes, we need to set up the pipe
		if (i < chain.operators().size() && chain.getOperator(i) == TokenType::Pipe)
		{
			// Find the end of this pipeline
			size_t pipelineEnd = i;
			while (pipelineEnd < chain.operators().size() && chain.getOperator(pipelineEnd) == TokenType::Pipe)
			{
				pipelineEnd++;
			}

			// Extract the pipeline commands
			std::vector<Command> pipelineCommands;
			for (size_t j = i; j <= pipelineEnd; ++j)
			{
				pipelineCommands.push_back(chain.getCommand(j));
			}

			// Execute the pipeline asynchronously (directly handle piped commands)
			lastExitStatus = co_await executeCommandsWithPipes(pipelineCommands);

			// Skip to after this pipeline
			i = pipelineEnd;

			// Check if we need to stop based on the previous operator
			if (i > 0 && i - 1 < chain.operators().size())
			{
				if (chain.getOperator(i - 1) == TokenType::And && lastExitStatus != 0)
				{
					// Stop if previous command failed for AND
					break;
				}
				else if (chain.getOperator(i - 1) == TokenType::Or && lastExitStatus == 0)
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
			if (i > 0 && i - 1 < chain.operators().size())
			{
				if (chain.getOperator(i - 1) == TokenType::And && lastExitStatus != 0)
				{
					// Skip this command if the previous one failed for AND
					if (debugMode)
					{
						std::cerr << "Debug: Skipping command due to AND operator and previous command failure" << std::endl;
					}
					continue;
				}
				else if (chain.getOperator(i - 1) == TokenType::Or && lastExitStatus == 0)
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
			bool result = co_await chain.getCommand(i).executeAsync();
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
		bool result = co_await commands[0].executeAsync();
		co_return result ? 0 : 1; // Convert bool to exit status
	}

	// For multiple commands, we need to set up pipes
	int lastExitStatus = 0;
	
	static_assert(false, "TODO:Need to implement the Source/Sink versions for proper piping");
	static_assert(false, "TODO:Need to implement the Source/Sink versions for proper piping");
	// For now, just execute each command sequentially as a workaround
	for (size_t i = 0; i < commands.size(); ++i)
	{
		debugInfo.commandCount++;
		bool result = co_await commands[i].executeAsync();
		lastExitStatus = result ? 0 : 1;
	}

	co_return lastExitStatus;
}
