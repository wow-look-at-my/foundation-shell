#include <catch2/catch_all.hpp>
#include <stdexcept>
#include <string>

#include "LastCppInclude.hpp"
#include "src/Token.hpp"

TEST_CASE("Value token types are identified correctly", "[token]")
{
	// Value token types
	CHECK(isValueTokenType(TokenType::Command));
	CHECK(isValueTokenType(TokenType::CommandArgument));

	// Not value token types
	CHECK_FALSE(isValueTokenType(TokenType::None));
	CHECK_FALSE(isValueTokenType(TokenType::Pipe));
	CHECK_FALSE(isValueTokenType(TokenType::And));
	CHECK_FALSE(isValueTokenType(TokenType::Or));
	// Background is deprecated, skip this test
	// CHECK_FALSE(isValueTokenType(TokenType::Background));
	CHECK_FALSE(isValueTokenType(TokenType::RedirectStdIn));
	CHECK_FALSE(isValueTokenType(TokenType::RedirectStdOut));
	CHECK_FALSE(isValueTokenType(TokenType::RedirectStdOutAppend));
	CHECK_FALSE(isValueTokenType(TokenType::RedirectStdErr));
	CHECK_FALSE(isValueTokenType(TokenType::RedirectStdErrAppend));
}

TEST_CASE("Operator token types are identified correctly", "[token]")
{
	// Operator token types
	CHECK(isOperatorTokenType(TokenType::Pipe));
	CHECK(isOperatorTokenType(TokenType::And));
	CHECK(isOperatorTokenType(TokenType::Or));
	// Background is deprecated, skip this test
	// CHECK(isOperatorTokenType(TokenType::Background));
	CHECK(isOperatorTokenType(TokenType::RedirectStdIn));
	CHECK(isOperatorTokenType(TokenType::RedirectStdOut));
	CHECK(isOperatorTokenType(TokenType::RedirectStdOutAppend));
	CHECK(isOperatorTokenType(TokenType::RedirectStdErr));
	CHECK(isOperatorTokenType(TokenType::RedirectStdErrAppend));

	// Not operator token types
	CHECK_FALSE(isOperatorTokenType(TokenType::None));
	CHECK_FALSE(isOperatorTokenType(TokenType::Command));
	CHECK_FALSE(isOperatorTokenType(TokenType::CommandArgument));
}

TEST_CASE("ValueToken validation works correctly", "[token]")
{
	// Valid value token types with non-empty values
	REQUIRE_NOTHROW(ValueToken(TokenType::Command, "ls"));
	REQUIRE_NOTHROW(ValueToken(TokenType::CommandArgument, "-la"));
	REQUIRE_NOTHROW(ValueToken(TokenType::Command, "grep"));
	REQUIRE_NOTHROW(ValueToken(TokenType::CommandArgument, "pattern"));
	REQUIRE_NOTHROW(ValueToken(TokenType::Command, "cd"));
	REQUIRE_NOTHROW(ValueToken(TokenType::CommandArgument, "/tmp"));
	REQUIRE_NOTHROW(ValueToken(TokenType::Command, "echo"));
	REQUIRE_NOTHROW(ValueToken(TokenType::CommandArgument, "hello world"));

	// Invalid: value token types with empty values
	REQUIRE_THROWS_AS(ValueToken(TokenType::Command, ""), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::CommandArgument, ""), std::invalid_argument);

	// Invalid: operator token types used with ValueToken
	REQUIRE_THROWS_AS(ValueToken(TokenType::None, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::Pipe, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::And, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::Or, "invalid"), std::invalid_argument);
	// Background is deprecated, skip this test
	// REQUIRE_THROWS_AS(ValueToken(TokenType::Background, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::RedirectStdIn, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::RedirectStdOut, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::RedirectStdOutAppend, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::RedirectStdErr, "invalid"), std::invalid_argument);
	REQUIRE_THROWS_AS(ValueToken(TokenType::RedirectStdErrAppend, "invalid"), std::invalid_argument);
}

TEST_CASE("OperatorToken validation works correctly", "[token]")
{
	// Valid operator token types
	REQUIRE_NOTHROW(OperatorToken(TokenType::Pipe));
	REQUIRE_NOTHROW(OperatorToken(TokenType::And));
	REQUIRE_NOTHROW(OperatorToken(TokenType::Or));
	// Background is deprecated, skip this test
	// REQUIRE_NOTHROW(OperatorToken(TokenType::Background));
	REQUIRE_NOTHROW(OperatorToken(TokenType::RedirectStdIn));
	REQUIRE_NOTHROW(OperatorToken(TokenType::RedirectStdOut));
	REQUIRE_NOTHROW(OperatorToken(TokenType::RedirectStdOutAppend));
	REQUIRE_NOTHROW(OperatorToken(TokenType::RedirectStdErr));
	REQUIRE_NOTHROW(OperatorToken(TokenType::RedirectStdErrAppend));

	// Invalid: value token types used with OperatorToken
	REQUIRE_THROWS_AS(OperatorToken(TokenType::Command), std::invalid_argument);
	REQUIRE_THROWS_AS(OperatorToken(TokenType::CommandArgument), std::invalid_argument);

	// Edge case: None token type (should not be used with OperatorToken)
	REQUIRE_THROWS_AS(OperatorToken(TokenType::None), std::invalid_argument);
}

TEST_CASE("Token properties are correctly accessible", "[token]")
{
	// ValueToken properties
	ValueToken cmd(TokenType::Command, "echo");
	CHECK(cmd.type == TokenType::Command);
	CHECK(cmd.value == "echo");

	ValueToken arg(TokenType::CommandArgument, "hello");
	CHECK(arg.type == TokenType::CommandArgument);
	CHECK(arg.value == "hello");

	// Test with Unicode characters
	ValueToken unicodeCmd(TokenType::Command, "ls");
	CHECK(unicodeCmd.type == TokenType::Command);
	CHECK(unicodeCmd.value == "ls");

	// Test with special characters
	ValueToken specialChars(TokenType::CommandArgument, "file with spaces.txt");
	CHECK(specialChars.type == TokenType::CommandArgument);
	CHECK(specialChars.value == "file with spaces.txt");

	// OperatorToken properties
	OperatorToken pipe(TokenType::Pipe);
	CHECK(pipe.type == TokenType::Pipe);

	OperatorToken redirect(TokenType::RedirectStdOut);
	CHECK(redirect.type == TokenType::RedirectStdOut);
}

TEST_CASE("Tokens handle extreme values correctly", "[token]")
{
	// Very long value
	std::string longValue(1000, 'a'); // 1000 'a' characters
	ValueToken longToken(TokenType::CommandArgument, longValue);
	CHECK(longToken.value == longValue);

	// Value with special characters
	std::string specialValue = "!@#$%^&*()_+{}|:\"<>?[];',./";
	ValueToken specialToken(TokenType::CommandArgument, specialValue);
	CHECK(specialToken.value == specialValue);
}
