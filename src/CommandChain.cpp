#include "CommandChain.hpp"
#include "Command.hpp"
#include "Token.hpp"
#include <iostream>
#include <format>
#include <vector>
#include <stdexcept>

// External globals from config.hpp - declared in other files
extern bool debugMode;
extern DebugInfo debugInfo;

// Constructor that parses tokens into a command chain
CommandChain::CommandChain(const std::vector<std::string> &tokens)
{
	// Use our static parsing method
	*this = parseFromTokens(tokens);
}

// Single command constructor
CommandChain::CommandChain(Command firstCommand)
{
	if (firstCommand.args.empty())
	{
		throw std::invalid_argument("Cannot add empty command to chain");
	}
	commands_.push_back(std::move(firstCommand));
}

// Append command with operator
void CommandChain::appendCommand(Command nextCommand, TokenType op)
{
	if (commands_.empty())
	{
		throw std::logic_error("Cannot append command to empty chain - use the constructor with a command first");
	}
	if (nextCommand.args.empty())
	{
		throw std::invalid_argument("Cannot add empty command to chain");
	}

	operators_.push_back(op);
	commands_.push_back(std::move(nextCommand));
}

// Helper function to convert a string token to our Token types
static std::variant<ValueToken, OperatorToken> parseStringToToken(const std::string &tokenStr, bool isLastToken)
{
	// Map string to token type and create the appropriate token
	if (tokenStr == "|")
	{
		return OperatorToken(TokenType::Pipe);
	}
	else if (tokenStr == "&&")
	{
		return OperatorToken(TokenType::And);
	}
	else if (tokenStr == "||")
	{
		return OperatorToken(TokenType::Or);
	}
	else if (tokenStr == "&" && isLastToken)
	{
		return OperatorToken(TokenType::Background);
	}
	else if (tokenStr == "<")
	{
		return OperatorToken(TokenType::RedirectStdIn);
	}
	else if (tokenStr == ">")
	{
		return OperatorToken(TokenType::RedirectStdOut);
	}
	else if (tokenStr == ">>")
	{
		return OperatorToken(TokenType::RedirectStdOutAppend);
	}
	else if (tokenStr == "2>")
	{
		return OperatorToken(TokenType::RedirectStdErr);
	}
	else if (tokenStr == "2>>")
	{
		return OperatorToken(TokenType::RedirectStdErrAppend);
	}
	else if (!tokenStr.empty())
	{
		// Default to command or argument token based on position
		static bool isFirstArg = true;
		TokenType type;

		if (isFirstArg)
		{
			type = TokenType::Command;
			isFirstArg = false;
		}
		else
		{
			type = TokenType::CommandArgument;
		}

		return ValueToken(type, tokenStr);
	}
	else
	{
		// Empty tokens are invalid, but we return a placeholder to prevent crashes
		throw std::invalid_argument("Empty token is not allowed");
	}
}

// Static method to parse tokens into a command chain
CommandChain CommandChain::parseFromTokens(const std::vector<std::string> &tokens)
{
	CommandChain chain;

	// Convert string tokens to our Token types with proper validation
	std::vector<std::variant<ValueToken, OperatorToken>> parsedTokens;
	parsedTokens.reserve(tokens.size());

	// First pass: Parse all tokens into our validated Token types
	for (size_t i = 0; i < tokens.size(); ++i)
	{
		bool isLastToken = (i == tokens.size() - 1);
		try
		{
			parsedTokens.push_back(parseStringToToken(tokens[i], isLastToken));
		}
		catch (const std::invalid_argument &e)
		{
			// Skip invalid tokens, but log them in debug mode
			if (debugMode)
			{
				std::cerr << std::format("Debug: Invalid token: {} - {}", tokens[i], e.what()) << std::endl;
			}
		}
	}

	// Second pass: Process the validated tokens into commands and operators
	Command currentCommand;
	bool resetCommand = false;

	for (size_t i = 0; i < parsedTokens.size(); ++i)
	{
		const auto &token = parsedTokens[i];

		// Handle the token based on its type
		if (std::holds_alternative<OperatorToken>(token))
		{
			const auto &opToken = std::get<OperatorToken>(token);
			TokenType tokenType = opToken.type;

			// Process operators
			if (tokenType == TokenType::Pipe ||
				tokenType == TokenType::And ||
				tokenType == TokenType::Or)
			{

				// Add the current command to the chain if it's not empty
				if (!currentCommand.args.empty())
				{
					if (chain.commands_.empty())
					{
						// First command in chain
						chain.commands_.push_back(std::move(currentCommand));
					}
					else
					{
						// Add command with operator
						chain.operators_.push_back(tokenType);
						chain.commands_.push_back(std::move(currentCommand));
					}
					currentCommand = Command(); // Reset for next command
					resetCommand = true;
				}
			}
			else if (tokenType == TokenType::Background)
			{
				// Background process - mark the current command
				currentCommand.backgroundProcess = true;

				// Add the command to the chain if it's not empty
				if (!currentCommand.args.empty())
				{
					if (chain.commands_.empty())
					{
						// First command in chain
						chain.commands_.push_back(std::move(currentCommand));
					}
					else
					{
						// Add with the appropriate operator
						chain.operators_.push_back(TokenType::None);
						chain.commands_.push_back(std::move(currentCommand));
					}
					resetCommand = true;
				}
			}
			else if (tokenType == TokenType::RedirectStdIn)
			{
				// Input redirection
				if (i + 1 < parsedTokens.size() &&
					std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto &valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.inputFile = valueToken.value;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdOut)
			{
				// Output redirection
				if (i + 1 < parsedTokens.size() &&
					std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto &valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.outputFile = valueToken.value;
					currentCommand.appendOutput = false;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdOutAppend)
			{
				// Output redirection (append)
				if (i + 1 < parsedTokens.size() &&
					std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto &valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.outputFile = valueToken.value;
					currentCommand.appendOutput = true;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdErr)
			{
				// Error redirection
				if (i + 1 < parsedTokens.size() &&
					std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto &valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.errorFile = valueToken.value;
					currentCommand.appendError = false;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdErrAppend)
			{
				// Error redirection (append)
				if (i + 1 < parsedTokens.size() &&
					std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto &valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.errorFile = valueToken.value;
					currentCommand.appendError = true;
					i++; // Skip the filename token
				}
			}
		}
		else if (std::holds_alternative<ValueToken>(token))
		{
			const auto &valueToken = std::get<ValueToken>(token);

			// Add command or argument to the current command
			currentCommand.args.push_back(valueToken.value);
			resetCommand = false;
		}
	}

	// Add any remaining command if it's not empty and we didn't just reset it
	if (!currentCommand.args.empty() && !resetCommand)
	{
		if (chain.commands_.empty())
		{
			chain.commands_.push_back(std::move(currentCommand));
		}
		else
		{
			// If we have multiple commands but no explicit operator, use None
			chain.operators_.push_back(TokenType::None);
			chain.commands_.push_back(std::move(currentCommand));
		}
	}

	// Validate the chain invariant: operators.size() == commands.size() - 1 or empty chain
	if (!chain.commands_.empty() && chain.operators_.size() != chain.commands_.size() - 1)
	{
		throw std::logic_error(std::format(
			"Invalid command chain: {} commands should have {} operators, but has {}",
			chain.commands_.size(), chain.commands_.size() - 1, chain.operators_.size()));
	}

	return chain;
}

// Implementation of the private method to execute piped commands
Task<int> CommandChain::executeCommandsWithPipesAsync(const std::vector<Command> &commands) const
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

	// Temporary implementation - should be replaced with proper Source/Sink
	// implementations in the future
	static_assert(false, "TODO:Implement full piping with proper Source/Sink interfaces");
	static_assert(false, "TODO:Implement full piping with proper Source/Sink interfaces");
	// For now, we'll execute commands sequentially as a placeholder

	// For now, just execute each command sequentially as a placeholder
	for (size_t i = 0; i < commands.size(); ++i)
	{
		debugInfo.commandCount++;
		bool result = co_await commands[i].executeAsync();
		lastExitStatus = result ? 0 : 1;
	}

	co_return lastExitStatus;
}

// Execute the command chain asynchronously
Task<int> CommandChain::executeAsync(const ShellConfig &config) const
{
	if (commands_.empty())
	{
		co_return 0;
	}

	if (debugMode)
	{
		std::cerr << "Debug: Executing command chain with " << commands_.size() << " commands" << std::endl;
		debugInfo.pipelineCount++;
	}

	// If there's only one command, execute it directly
	if (commands_.size() == 1)
	{
		if (debugMode)
		{
			std::cerr << "Debug: Executing single command: " << commands_[0].args[0] << std::endl;

			// Count redirections
			if (!commands_[0].inputFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Input redirection from " << commands_[0].inputFile << std::endl;
			}
			if (!commands_[0].outputFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Output redirection to " << commands_[0].outputFile
						  << (commands_[0].appendOutput ? " (append)" : "") << std::endl;
			}
			if (!commands_[0].errorFile.empty())
			{
				debugInfo.redirectionCount++;
				std::cerr << "Debug: Error redirection to " << commands_[0].errorFile
						  << (commands_[0].appendError ? " (append)" : "") << std::endl;
			}

			// Count background processes
			if (commands_[0].backgroundProcess)
			{
				debugInfo.backgroundProcessCount++;
				std::cerr << "Debug: Running as background process" << std::endl;
			}
		}

		debugInfo.commandCount++;
		bool result = co_await commands_[0].executeAsync();
		co_return result ? 0 : 1; // Convert bool to exit status
	}

	// For multiple commands, handle the command chain
	int lastExitStatus = 0;

	for (size_t i = 0; i < commands_.size(); ++i)
	{
		// For pipes, we need to set up the pipe
		if (i < operators_.size() && operators_[i] == TokenType::Pipe)
		{
			// Find the end of this pipeline
			size_t pipelineEnd = i;
			while (pipelineEnd < operators_.size() && operators_[pipelineEnd] == TokenType::Pipe)
			{
				pipelineEnd++;
			}

			// Extract the pipeline commands
			std::vector<Command> pipelineCommands;
			for (size_t j = i; j <= pipelineEnd; ++j)
			{
				pipelineCommands.push_back(commands_[j]);
			}

			// Execute the pipeline asynchronously (directly handle piped commands)
			lastExitStatus = co_await executeCommandsWithPipesAsync(pipelineCommands);

			// Skip to after this pipeline
			i = pipelineEnd;

			// Check if we need to stop based on the previous operator
			if (i > 0 && i - 1 < operators_.size())
			{
				if (operators_[i - 1] == TokenType::And && lastExitStatus != 0)
				{
					// Stop if previous command failed for AND
					break;
				}
				else if (operators_[i - 1] == TokenType::Or && lastExitStatus == 0)
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
			if (i > 0 && i - 1 < operators_.size())
			{
				if (operators_[i - 1] == TokenType::And && lastExitStatus != 0)
				{
					// Skip this command if the previous one failed for AND
					if (debugMode)
					{
						std::cerr << "Debug: Skipping command due to AND operator and previous command failure" << std::endl;
					}
					continue;
				}
				else if (operators_[i - 1] == TokenType::Or && lastExitStatus == 0)
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
			bool result = co_await commands_[i].executeAsync();
			lastExitStatus = result ? 0 : 1; // Convert bool to exit status
		}
	}

	co_return lastExitStatus;
}
