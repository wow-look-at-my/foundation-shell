#pragma once

#include <string>
#include <string_view>
#include <variant>
#include <stdexcept>

// Token types for lexical analysis
enum class TokenType
{
    None,                       // No operator
    Command,                    // Command name (first arg)
    CommandArgument,            // Command arguments
    Pipe,                       // | (pipe)
    And,                        // && (and)
    Or,                         // || (or)
    Background [[deprecated]],  // & (background)
    RedirectStdIn,              // < (input redirection)
    RedirectStdOut,             // > (output redirection)
    RedirectStdOutAppend,       // >> (append redirection)
    RedirectStdErr,             // 2> (error redirection)
    RedirectStdErrAppend,       // 2>> (error append redirection)
};

// Helper functions to validate token types
inline bool isValueTokenType(TokenType type) {
    return type == TokenType::Command || type == TokenType::CommandArgument;
}

inline bool isOperatorTokenType(TokenType type) {
    return type == TokenType::Pipe || 
           type == TokenType::And || 
           type == TokenType::Or || 
           type == TokenType::Background || 
           type == TokenType::RedirectStdIn || 
           type == TokenType::RedirectStdOut || 
           type == TokenType::RedirectStdOutAppend || 
           type == TokenType::RedirectStdErr || 
           type == TokenType::RedirectStdErrAppend;
}

// Base token type (non-virtual for simplicity)
struct Token {
    TokenType type;
    
    explicit Token(TokenType t) : type(t) {}
};

// Value token (stores a string value)
struct ValueToken : Token {
    std::string value;
    
    ValueToken(TokenType t, std::string_view v) : Token(t), value(v) {
        if (!isValueTokenType(t)) {
            throw std::invalid_argument("ValueToken must be constructed with a value token type");
        }
        if (v.empty()) {
            throw std::invalid_argument("ValueToken cannot have an empty value");
        }
    }
};

// Operator token (no value, just the type)
struct OperatorToken : Token {
    explicit OperatorToken(TokenType t) : Token(t) {
        if (!isOperatorTokenType(t)) {
            throw std::invalid_argument("OperatorToken must be constructed with an operator token type");
        }
    }
};