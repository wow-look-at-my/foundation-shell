#pragma once

// Token types for lexical analysis
enum class TokenType
{
	None,                 // No operator
	Command,              // Command name (first arg)
	CommandArgument,      // Command arguments
	Pipe,                 // | (pipe)
	And,                  // && (and)
	Or,                   // || (or)
	RedirectStdIn,        // < (input redirection)
	RedirectStdOut,       // > (output redirection)
	RedirectStdOutAppend, // >> (append redirection)
	RedirectStdErr,       // 2> (error redirection)
	RedirectStdErrAppend, // 2>> (error append redirection)
};

// Helper functions to validate token types
inline bool isValueTokenType(TokenType type)
{
	return type == TokenType::Command || type == TokenType::CommandArgument;
}

inline bool isOperatorTokenType(TokenType type)
{
	return type == TokenType::Pipe || type == TokenType::And || type == TokenType::Or ||
	       type == TokenType::RedirectStdIn || type == TokenType::RedirectStdOut ||
	       type == TokenType::RedirectStdOutAppend || type == TokenType::RedirectStdErr ||
	       type == TokenType::RedirectStdErrAppend;
}
