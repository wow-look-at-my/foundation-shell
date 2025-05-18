#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "../src/Token.hpp"
#include <stdexcept>
#include <string>

// Test that value token types are identified correctly
TEST(TokenTest, ValueTokenTypeIdentification)
{
    // Value token types
    EXPECT_TRUE(isValueTokenType(TokenType::Command));
    EXPECT_TRUE(isValueTokenType(TokenType::CommandArgument));
    
    // Not value token types
    EXPECT_FALSE(isValueTokenType(TokenType::None));
    EXPECT_FALSE(isValueTokenType(TokenType::Pipe));
    EXPECT_FALSE(isValueTokenType(TokenType::And));
    EXPECT_FALSE(isValueTokenType(TokenType::Or));
    EXPECT_FALSE(isValueTokenType(TokenType::Background));
    EXPECT_FALSE(isValueTokenType(TokenType::RedirectStdIn));
    EXPECT_FALSE(isValueTokenType(TokenType::RedirectStdOut));
    EXPECT_FALSE(isValueTokenType(TokenType::RedirectStdOutAppend));
    EXPECT_FALSE(isValueTokenType(TokenType::RedirectStdErr));
    EXPECT_FALSE(isValueTokenType(TokenType::RedirectStdErrAppend));
}

// Test that operator token types are identified correctly
TEST(TokenTest, OperatorTokenTypeIdentification)
{
    // Operator token types
    EXPECT_TRUE(isOperatorTokenType(TokenType::Pipe));
    EXPECT_TRUE(isOperatorTokenType(TokenType::And));
    EXPECT_TRUE(isOperatorTokenType(TokenType::Or));
    EXPECT_TRUE(isOperatorTokenType(TokenType::Background));
    EXPECT_TRUE(isOperatorTokenType(TokenType::RedirectStdIn));
    EXPECT_TRUE(isOperatorTokenType(TokenType::RedirectStdOut));
    EXPECT_TRUE(isOperatorTokenType(TokenType::RedirectStdOutAppend));
    EXPECT_TRUE(isOperatorTokenType(TokenType::RedirectStdErr));
    EXPECT_TRUE(isOperatorTokenType(TokenType::RedirectStdErrAppend));
    
    // Not operator token types
    EXPECT_FALSE(isOperatorTokenType(TokenType::None));
    EXPECT_FALSE(isOperatorTokenType(TokenType::Command));
    EXPECT_FALSE(isOperatorTokenType(TokenType::CommandArgument));
}

// Test all valid and invalid ValueToken constructions
TEST(TokenTest, ValueTokenValidations)
{
    // Valid value token types with non-empty values
    EXPECT_NO_THROW(ValueToken(TokenType::Command, "ls"));
    EXPECT_NO_THROW(ValueToken(TokenType::CommandArgument, "-la"));
    EXPECT_NO_THROW(ValueToken(TokenType::Command, "grep"));
    EXPECT_NO_THROW(ValueToken(TokenType::CommandArgument, "pattern"));
    EXPECT_NO_THROW(ValueToken(TokenType::Command, "cd"));
    EXPECT_NO_THROW(ValueToken(TokenType::CommandArgument, "/tmp"));
    EXPECT_NO_THROW(ValueToken(TokenType::Command, "echo"));
    EXPECT_NO_THROW(ValueToken(TokenType::CommandArgument, "hello world"));
    
    // Invalid: value token types with empty values
    EXPECT_THROW(ValueToken(TokenType::Command, ""), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::CommandArgument, ""), std::invalid_argument);
    
    // Invalid: operator token types used with ValueToken
    EXPECT_THROW(ValueToken(TokenType::None, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::Pipe, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::And, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::Or, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::Background, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::RedirectStdIn, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::RedirectStdOut, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::RedirectStdOutAppend, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::RedirectStdErr, "invalid"), std::invalid_argument);
    EXPECT_THROW(ValueToken(TokenType::RedirectStdErrAppend, "invalid"), std::invalid_argument);
}

// Test all valid and invalid OperatorToken constructions
TEST(TokenTest, OperatorTokenValidations)
{
    // Valid operator token types
    EXPECT_NO_THROW(OperatorToken(TokenType::Pipe));
    EXPECT_NO_THROW(OperatorToken(TokenType::And));
    EXPECT_NO_THROW(OperatorToken(TokenType::Or));
    EXPECT_NO_THROW(OperatorToken(TokenType::Background));
    EXPECT_NO_THROW(OperatorToken(TokenType::RedirectStdIn));
    EXPECT_NO_THROW(OperatorToken(TokenType::RedirectStdOut));
    EXPECT_NO_THROW(OperatorToken(TokenType::RedirectStdOutAppend));
    EXPECT_NO_THROW(OperatorToken(TokenType::RedirectStdErr));
    EXPECT_NO_THROW(OperatorToken(TokenType::RedirectStdErrAppend));
    
    // Invalid: value token types used with OperatorToken
    EXPECT_THROW(OperatorToken(TokenType::Command), std::invalid_argument);
    EXPECT_THROW(OperatorToken(TokenType::CommandArgument), std::invalid_argument);
    
    // Edge case: None token type (should not be used with OperatorToken)
    EXPECT_THROW(OperatorToken(TokenType::None), std::invalid_argument);
}

// Test token property access and behavior
TEST(TokenTest, TokenProperties)
{
    // ValueToken properties
    ValueToken cmd(TokenType::Command, "echo");
    EXPECT_EQ(cmd.type, TokenType::Command);
    EXPECT_EQ(cmd.value, "echo");
    
    ValueToken arg(TokenType::CommandArgument, "hello");
    EXPECT_EQ(arg.type, TokenType::CommandArgument);
    EXPECT_EQ(arg.value, "hello");
    
    // Test with Unicode characters
    ValueToken unicodeCmd(TokenType::Command, "ls");
    EXPECT_EQ(unicodeCmd.type, TokenType::Command);
    EXPECT_EQ(unicodeCmd.value, "ls");
    
    // Test with special characters
    ValueToken specialChars(TokenType::CommandArgument, "file with spaces.txt");
    EXPECT_EQ(specialChars.type, TokenType::CommandArgument);
    EXPECT_EQ(specialChars.value, "file with spaces.txt");
    
    // OperatorToken properties
    OperatorToken pipe(TokenType::Pipe);
    EXPECT_EQ(pipe.type, TokenType::Pipe);
    
    OperatorToken redirect(TokenType::RedirectStdOut);
    EXPECT_EQ(redirect.type, TokenType::RedirectStdOut);
}

// Test token with extreme values
TEST(TokenTest, TokenExtremeValues)
{
    // Very long value
    std::string longValue(1000, 'a'); // 1000 'a' characters
    ValueToken longToken(TokenType::CommandArgument, longValue);
    EXPECT_EQ(longToken.value, longValue);
    
    // Value with special characters
    std::string specialValue = "!@#$%^&*()_+{}|:\"<>?[];',./";
    ValueToken specialToken(TokenType::CommandArgument, specialValue);
    EXPECT_EQ(specialToken.value, specialValue);
}

// Main function provided by gtest_main library
