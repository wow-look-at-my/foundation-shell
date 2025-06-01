#include "CommandChain.hpp"

#include <atomic>
#include <cstdio>
#include <format>
#include <future>
#include <iostream>
#include <mh/concurrency/dispatcher.hpp>
#include <stdexcept>
#include <thread>
#include <vector>

#include "Command.hpp"
#include "LastCppInclude.hpp"
#include "Token.hpp"
#include "io/IPipe.hpp" // For IPipe

// Constructor that parses tokens into a command chain
CommandChain::CommandChain(const std::vector<std::string>& tokens)
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

enum class TokenPosition
{
	FirstInCommand,
	SubsequentInCommand
};

// Helper function to convert a string token to our Token types
static std::variant<ValueToken, OperatorToken> parseStringToToken(const std::string& tokenStr, TokenPosition position)
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
		TokenType type = (position == TokenPosition::FirstInCommand) ? TokenType::Command : TokenType::CommandArgument;
		return ValueToken(type, tokenStr);
	}
	else
	{
		// Empty tokens are invalid, but we return a placeholder to prevent crashes
		throw std::invalid_argument("Empty token is not allowed");
	}
}

// Static method to parse tokens into a command chain
CommandChain CommandChain::parseFromTokens(const std::vector<std::string>& tokens)
{
	CommandChain chain;

	// Convert string tokens to our Token types with proper validation
	std::vector<std::variant<ValueToken, OperatorToken>> parsedTokens;
	parsedTokens.reserve(tokens.size());

	// First pass: Parse all tokens into our validated Token types
	bool isFirstInCommand = true;
	for (size_t i = 0; i < tokens.size(); ++i)
	{
		//		try
		//		{
		TokenPosition position = isFirstInCommand ? TokenPosition::FirstInCommand : TokenPosition::SubsequentInCommand;
		auto token = parseStringToToken(tokens[i], position);
		parsedTokens.push_back(token);

		// Reset position tracking when we hit an operator that separates commands
		if (std::holds_alternative<OperatorToken>(token))
		{
			const auto& opToken = std::get<OperatorToken>(token);
			if (opToken.type == TokenType::Pipe || opToken.type == TokenType::And || opToken.type == TokenType::Or)
			{
				isFirstInCommand = true;
			}
		}
		else if (isFirstInCommand)
		{
			// After the first value token in a command, subsequent ones are arguments
			isFirstInCommand = false;
		}
	}
	// catch (std::invalid_argument e)
	//{
	//	// Throw to bail out - invalid tokens should not be silently ignored
	//	std::print(stderr, "Invalid token '{}': {}\n", tokens[i], e.what());
	//	std::print(stderr, "Full token list: ");
	//	for (size_t j = 0; j < tokens.size(); ++j)
	//	{
	//		std::print(stderr, "'{}'{}", tokens[j], (j < tokens.size() - 1) ? ", " : "\n");
	//	}
	//	throw;
	// }

	// Second pass: Process the validated tokens into commands and operators
	Command currentCommand;
	bool resetCommand = false;

	for (size_t i = 0; i < parsedTokens.size(); ++i)
	{
		const auto& token = parsedTokens[i];

		// Handle the token based on its type
		if (std::holds_alternative<OperatorToken>(token))
		{
			const auto& opToken = std::get<OperatorToken>(token);
			TokenType tokenType = opToken.type;

			// Process operators
			if (tokenType == TokenType::Pipe || tokenType == TokenType::And || tokenType == TokenType::Or)
			{

				// Add the current command to the chain if it's not empty
				if (!currentCommand.args.empty())
				{
					// Add the current command first
					chain.commands_.push_back(std::move(currentCommand));

					// Only add the operator if there will be another command after this operator
					// We'll check this by looking ahead in the token stream
					bool hasCommandAfterOperator = false;
					for (size_t j = i + 1; j < parsedTokens.size(); ++j)
					{
						if (std::holds_alternative<ValueToken>(parsedTokens[j]))
						{
							hasCommandAfterOperator = true;
							break;
						}
						// Skip redirection operators, they don't start new commands
						if (std::holds_alternative<OperatorToken>(parsedTokens[j]))
						{
							const auto& nextOp = std::get<OperatorToken>(parsedTokens[j]);
							if (nextOp.type == TokenType::Pipe || nextOp.type == TokenType::And ||
							    nextOp.type == TokenType::Or)
							{
								break; // Another command operator means no command follows
							}
						}
					}

					// Only add the operator if there's a command after it
					if (hasCommandAfterOperator)
					{
						chain.operators_.push_back(tokenType);
					}

					currentCommand = Command(); // Reset for next command
					resetCommand = true;
				}
			}
			else if (tokenType == TokenType::RedirectStdIn)
			{
				// Input redirection
				if (i + 1 < parsedTokens.size() && std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto& valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.inputFile = valueToken.value;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdOut)
			{
				// Output redirection
				if (i + 1 < parsedTokens.size() && std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto& valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.outputFile = valueToken.value;
					currentCommand.appendOutput = false;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdOutAppend)
			{
				// Output redirection (append)
				if (i + 1 < parsedTokens.size() && std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto& valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.outputFile = valueToken.value;
					currentCommand.appendOutput = true;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdErr)
			{
				// Error redirection
				if (i + 1 < parsedTokens.size() && std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto& valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.errorFile = valueToken.value;
					currentCommand.appendError = false;
					i++; // Skip the filename token
				}
			}
			else if (tokenType == TokenType::RedirectStdErrAppend)
			{
				// Error redirection (append)
				if (i + 1 < parsedTokens.size() && std::holds_alternative<ValueToken>(parsedTokens[i + 1]))
				{
					const auto& valueToken = std::get<ValueToken>(parsedTokens[i + 1]);
					currentCommand.errorFile = valueToken.value;
					currentCommand.appendError = true;
					i++; // Skip the filename token
				}
			}
		}
		else if (std::holds_alternative<ValueToken>(token))
		{
			const auto& valueToken = std::get<ValueToken>(token);

			// Add command or argument to the current command
			currentCommand.args.push_back(valueToken.value);
			resetCommand = false;
		}
	}

	// Add any remaining command if it's not empty and we didn't just reset it
	if (!currentCommand.args.empty() && !resetCommand)
	{
		chain.commands_.push_back(std::move(currentCommand));
	}

	// Validate the chain invariant: operators.size() == commands.size() - 1 or empty chain
	if (!chain.commands_.empty() && chain.operators_.size() != chain.commands_.size() - 1)
	{
		throw std::logic_error(std::format("Invalid command chain: {} commands should have {} operators, but has {}",
		                                   chain.commands_.size(), chain.commands_.size() - 1,
		                                   chain.operators_.size()));
	}

	return chain;
}


// Execute the command chain asynchronously
mh::task<int> CommandChain::executeAsync() const
{
	if (commands_.empty())
	{
		co_return 0;
	}

	// Log message about executing command chain
	// std::cerr << "Debug: Executing command chain with " << commands_.size() << " commands" << std::endl;

	// If there's only one command, execute it directly
	if (commands_.size() == 1)
	{
		// Log single command execution info
		// std::cerr << "Debug: Executing single command: " << commands_[0].args[0] << std::endl;

		// Log redirections
		if (!commands_[0].inputFile.empty())
		{
			// std::cerr << "Debug: Input redirection from " << commands_[0].inputFile << std::endl;
		}
		if (!commands_[0].outputFile.empty())
		{
			// std::cerr << "Debug: Output redirection to " << commands_[0].outputFile
			//		  << (commands_[0].appendOutput ? " (append)" : "") << std::endl;
		}
		if (!commands_[0].errorFile.empty())
		{
			// std::cerr << "Debug: Error redirection to " << commands_[0].errorFile
			//		  << (commands_[0].appendError ? " (append)" : "") << std::endl;
		}

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

			// Execute the pipeline asynchronously with concurrent commands
			if (pipelineCommands.size() == 1)
			{
				bool result = co_await pipelineCommands[0].executeAsync();
				lastExitStatus = result ? 0 : 1;
			}
			else
			{
				// Create pipes between commands
				std::vector<std::shared_ptr<IPipe>> pipes;
				for (size_t k = 0; k < pipelineCommands.size() - 1; ++k)
				{
					pipes.emplace_back(IPipe::create());
				}

				// Start all commands concurrently
				std::vector<mh::task<bool>> tasks;
				tasks.reserve(pipelineCommands.size());

				for (size_t k = 0; k < pipelineCommands.size(); ++k)
				{
					if (k > 0 && k < pipelineCommands.size() - 1)
					{
						// Middle command: input from previous pipe, output to next pipe
						tasks.emplace_back(
						    pipelineCommands[k].executeAsync(pipes[k - 1]->getSource(), pipes[k]->getSink()));
					}
					else if (k > 0)
					{
						// Last command: input from previous pipe
						tasks.emplace_back(pipelineCommands[k].executeAsync(pipes[k - 1]->getSource()));
					}
					else
					{
						// First command: output to next pipe
						tasks.emplace_back(pipelineCommands[k].executeAsync(nullptr, pipes[k]->getSink()));
					}
				}

				// Wait for all commands to complete
				bool lastResult = true;
				for (size_t k = 0; k < tasks.size(); ++k)
				{
					bool result = co_await tasks[k];
					if (k == pipelineCommands.size() - 1)
					{
						lastResult = result;
					}
				}

				lastExitStatus = lastResult ? 0 : 1;
			}

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
			// Command executed

			// Check if we should execute this command based on the previous exit status
			if (i > 0 && i - 1 < operators_.size())
			{
				if (operators_[i - 1] == TokenType::And && lastExitStatus != 0)
				{
					// Skip this command if the previous one failed for AND
					// std::cerr << "Debug: Skipping command due to AND operator and previous command failure" <<
					// std::endl;
					continue;
				}
				else if (operators_[i - 1] == TokenType::Or && lastExitStatus == 0)
				{
					// Skip this command if the previous one succeeded for OR
					// std::cerr << "Debug: Skipping command due to OR operator and previous command success" <<
					// std::endl;
					continue;
				}
			}

			// Execute the command directly and asynchronously
			bool result = co_await commands_[i].executeAsync();
			lastExitStatus = result ? 0 : 1; // Convert bool to exit status

			// Report command failure with clear diagnostics
			if (!result)
			{
				std::string commandStr = commands_[i].args.empty() ? "<empty>" : commands_[i].args[0];
				for (size_t j = 1; j < commands_[i].args.size(); ++j)
				{
					commandStr += " " + commands_[i].args[j];
				}
				std::print(stderr, "{}Command failed: {}{}\n", Colors::COLOR_RED, commandStr, Colors::COLOR_RESET);
				std::print(stderr, "{}Exit status: {}{}\n", Colors::COLOR_RED, lastExitStatus, Colors::COLOR_RESET);

				// Show pointer to the failed command in the chain
				std::string chainStr;
				for (size_t k = 0; k < commands_.size(); ++k)
				{
					if (k > 0 && k - 1 < operators_.size())
					{
						if (operators_[k - 1] == TokenType::Pipe)
							chainStr += " | ";
						else if (operators_[k - 1] == TokenType::And)
							chainStr += " && ";
						else if (operators_[k - 1] == TokenType::Or)
							chainStr += " || ";
					}
					chainStr += commands_[k].args.empty() ? "<empty>" : commands_[k].args[0];
				}
				std::print(stderr, "{}Chain: {}{}\n", Colors::COLOR_YELLOW, chainStr, Colors::COLOR_RESET);

				// Add pointer to failed command
				std::string pointer;
				size_t pos = 0;
				for (size_t k = 0; k < i; ++k)
				{
					if (k > 0 && k - 1 < operators_.size())
					{
						if (operators_[k - 1] == TokenType::Pipe)
							pos += 3; // " | "
						else if (operators_[k - 1] == TokenType::And)
							pos += 4; // " && "
						else if (operators_[k - 1] == TokenType::Or)
							pos += 4; // " || "
					}
					pos += commands_[k].args.empty() ? 7 : commands_[k].args[0].length(); // "<empty>" or command length
				}
				pointer = std::string(pos, ' ') + std::string(commandStr.length(), '^');
				std::print(stderr, "{}       {}{}\n", Colors::COLOR_RED, pointer, Colors::COLOR_RESET);
			}
		}
	}

	co_return lastExitStatus;
}
