#pragma once

#include <string>
#include <string_view>
#include <variant>
#include <stdexcept>
#include "TokenType.hpp"

// Base token type (non-virtual for simplicity)
struct Token
{
	TokenType type;

	explicit Token(TokenType t) : type(t) {}
};

// Value token (stores a string value)
struct ValueToken : Token
{
	std::string value;

	ValueToken(TokenType t, std::string_view v) : Token(t), value(v)
	{
		if (!isValueTokenType(t))
		{
			throw std::invalid_argument("ValueToken must be constructed with a value token type");
		}
		if (v.empty())
		{
			throw std::invalid_argument("ValueToken cannot have an empty value");
		}
	}
};

// Operator token (no value, just the type)
struct OperatorToken : Token
{
	explicit OperatorToken(TokenType t) : Token(t)
	{
		if (!isOperatorTokenType(t))
		{
			throw std::invalid_argument("OperatorToken must be constructed with an operator token type");
		}
	}
};
