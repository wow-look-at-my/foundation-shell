#include <catch2/catch_all.hpp>
#include <stdexcept>
#include <string>
#include <vector>

#include "LastCppInclude.hpp"
#include "src/Command.hpp"
#include "src/CommandChain.hpp"
#include "src/Token.hpp"

// Test creating a CommandChain with valid commands
TEST_CASE("CommandChain valid construction", "[command_chain]")
{
	// Test creating empty chain
	REQUIRE_NOTHROW(CommandChain());
	CommandChain emptyChain;
	CHECK(emptyChain.empty());
	CHECK(emptyChain.size() == 0);

	// Test creating chain with a single command
	Command cmd;
	cmd.args = {"ls", "-la"};
	REQUIRE_NOTHROW(CommandChain(cmd));

	CommandChain singleCmdChain(cmd);
	CHECK_FALSE(singleCmdChain.empty());
	CHECK(singleCmdChain.size() == 1);
	CHECK(singleCmdChain.commands().size() == 1);
	CHECK(singleCmdChain.operators().size() == 0);
	CHECK(singleCmdChain.getCommand(0).args[0] == "ls");
	CHECK(singleCmdChain.getCommand(0).args[1] == "-la");

	// Test appending a command to the chain
	Command cmd2;
	cmd2.args = {"grep", "pattern"};
	REQUIRE_NOTHROW(singleCmdChain.appendCommand(cmd2, TokenType::Pipe));

	CHECK(singleCmdChain.size() == 2);
	CHECK(singleCmdChain.commands().size() == 2);
	CHECK(singleCmdChain.operators().size() == 1);
	CHECK(singleCmdChain.getOperator(0) == TokenType::Pipe);

	// Test appending more commands
	Command cmd3;
	cmd3.args = {"wc", "-l"};
	REQUIRE_NOTHROW(singleCmdChain.appendCommand(cmd3, TokenType::And));

	CHECK(singleCmdChain.size() == 3);
	CHECK(singleCmdChain.commands().size() == 3);
	CHECK(singleCmdChain.operators().size() == 2);
	CHECK(singleCmdChain.getOperator(1) == TokenType::And);

	// Verify invariant: operators.size() == commands.size() - 1
	CHECK(singleCmdChain.operators().size() == singleCmdChain.commands().size() - 1);
}

// Test invalid CommandChain constructions
TEST_CASE("CommandChain invalid construction", "[command_chain]")
{
	// Test creating chain with an empty command
	Command emptyCmd;
	REQUIRE_THROWS_AS(CommandChain(emptyCmd), std::invalid_argument);

	// Test appending to an empty chain
	CommandChain emptyChain;
	Command validCmd;
	validCmd.args = {"ls"};
	REQUIRE_THROWS_AS(emptyChain.appendCommand(validCmd, TokenType::Pipe), std::logic_error);

	// Test appending an empty command
	CommandChain chain(validCmd);
	Command emptyCmd2;
	REQUIRE_THROWS_AS(chain.appendCommand(emptyCmd2, TokenType::Pipe), std::invalid_argument);

	// Test out-of-bounds access
	REQUIRE_THROWS_AS(chain.getCommand(1), std::out_of_range);
	REQUIRE_THROWS_AS(chain.getOperator(0), std::out_of_range);

	// Test invalid token parsing
	std::vector<std::string> invalidTokens = {"", "|", ""};
	// Empty tokens should throw an exception as they're not valid
	REQUIRE_THROWS_AS(CommandChain::parseFromTokens(invalidTokens), std::invalid_argument);
}

// Test parsing from tokens
TEST_CASE("CommandChain parse from tokens", "[command_chain]")
{
	// Test parsing a simple command
	std::vector<std::string> simpleTokens = {"ls", "-la"};
	CommandChain simpleChain = CommandChain::parseFromTokens(simpleTokens);
	CHECK(simpleChain.size() == 1);
	CHECK(simpleChain.commands()[0].args[0] == "ls");
	CHECK(simpleChain.commands()[0].args[1] == "-la");

	// Test parsing a piped command
	std::vector<std::string> pipedTokens = {"ls", "-la", "|", "grep", "pattern"};
	CommandChain pipedChain = CommandChain::parseFromTokens(pipedTokens);
	CHECK(pipedChain.size() == 2);
	CHECK(pipedChain.operators()[0] == TokenType::Pipe);
	CHECK(pipedChain.commands()[0].args[0] == "ls");
	CHECK(pipedChain.commands()[1].args[0] == "grep");

	// Test parsing AND operator
	std::vector<std::string> andTokens = {"mkdir", "test", "&&", "cd", "test"};
	CommandChain andChain = CommandChain::parseFromTokens(andTokens);
	CHECK(andChain.size() == 2);
	CHECK(andChain.operators()[0] == TokenType::And);

	// Test parsing OR operator
	std::vector<std::string> orTokens = {"ls", "nonexistent", "||", "echo", "not found"};
	CommandChain orChain = CommandChain::parseFromTokens(orTokens);
	CHECK(orChain.size() == 2);
	CHECK(orChain.operators()[0] == TokenType::Or);

	// Test parsing with input redirection
	std::vector<std::string> inRedirTokens = {"sort", "<", "input.txt"};
	CommandChain inRedirChain = CommandChain::parseFromTokens(inRedirTokens);
	CHECK(inRedirChain.size() == 1);
	CHECK(inRedirChain.commands()[0].inputFile == "input.txt");

	// Test parsing with output redirection
	std::vector<std::string> outRedirTokens = {"ls", "-la", ">", "output.txt"};
	CommandChain outRedirChain = CommandChain::parseFromTokens(outRedirTokens);
	CHECK(outRedirChain.size() == 1);
	CHECK(outRedirChain.commands()[0].outputFile == "output.txt");
	CHECK_FALSE(outRedirChain.commands()[0].appendOutput);

	// Test parsing with append output redirection
	std::vector<std::string> appendRedirTokens = {"echo", "hello", ">>", "output.txt"};
	CommandChain appendRedirChain = CommandChain::parseFromTokens(appendRedirTokens);
	CHECK(appendRedirChain.size() == 1);
	CHECK(appendRedirChain.commands()[0].outputFile == "output.txt");
	CHECK(appendRedirChain.commands()[0].appendOutput);

	// Test parsing with error redirection
	std::vector<std::string> errRedirTokens = {"gcc", "file.c", "2>", "error.log"};
	CommandChain errRedirChain = CommandChain::parseFromTokens(errRedirTokens);
	CHECK(errRedirChain.size() == 1);
	CHECK(errRedirChain.commands()[0].errorFile == "error.log");
	CHECK_FALSE(errRedirChain.commands()[0].appendError);

	// Test parsing with append error redirection
	std::vector<std::string> appendErrRedirTokens = {"gcc", "file.c", "2>>", "error.log"};
	CommandChain appendErrRedirChain = CommandChain::parseFromTokens(appendErrRedirTokens);
	CHECK(appendErrRedirChain.size() == 1);
	CHECK(appendErrRedirChain.commands()[0].errorFile == "error.log");
	CHECK(appendErrRedirChain.commands()[0].appendError);

	// Skip background process test as it seems the Command class
	// does not have a backgroundProcess member

	// Test parsing complex command chain
	std::vector<std::string> complexTokens = {"grep", "error", "<",  "log.txt", "|",           "sort",
	                                          "|",    "uniq",  "-c", ">",       "results.txt", "&&",
	                                          "echo", "done",  "||", "echo",    "failed"};
	CommandChain complexChain = CommandChain::parseFromTokens(complexTokens);
	CHECK(complexChain.size() == 5);
	CHECK(complexChain.operators()[0] == TokenType::Pipe);
	CHECK(complexChain.operators()[1] == TokenType::Pipe);
	CHECK(complexChain.operators()[2] == TokenType::And);
	CHECK(complexChain.operators()[3] == TokenType::Or);
	CHECK(complexChain.commands()[0].inputFile == "log.txt");
	CHECK(complexChain.commands()[2].outputFile == "results.txt");

	// Verify invariant: operators.size() == commands.size() - 1
	CHECK(complexChain.operators().size() == complexChain.commands().size() - 1);
}

// Test edge cases
TEST_CASE("CommandChain edge cases", "[command_chain]")
{
	// Test empty tokens
	std::vector<std::string> emptyTokens = {};
	CommandChain emptyChain = CommandChain::parseFromTokens(emptyTokens);
	CHECK(emptyChain.empty());

	// Test tokens with only operators (invalid command but shouldn't crash)
	std::vector<std::string> onlyOperators = {"|", "&&", "||"};
	CommandChain onlyOperatorsChain = CommandChain::parseFromTokens(onlyOperators);
	CHECK(onlyOperatorsChain.empty());

	// Test missing redirection targets
	std::vector<std::string> missingTarget = {"cat", ">"};
	CommandChain missingTargetChain = CommandChain::parseFromTokens(missingTarget);
	CHECK(missingTargetChain.size() == 1);
	CHECK(missingTargetChain.commands()[0].outputFile.empty());

	// Test & not at the end
	std::vector<std::string> invalidBackground = {"ls", "&", "-la"};
	CommandChain invalidBackgroundChain = CommandChain::parseFromTokens(invalidBackground);
	CHECK(invalidBackgroundChain.size() == 1);
	// Skip this check as Command does not have backgroundProcess member
	// CHECK_FALSE(invalidBackgroundChain.commands()[0].backgroundProcess);
	CHECK(invalidBackgroundChain.commands()[0].args[1] == "&");
}

// Test invariant enforcement - making sure it's impossible to create an invalid state
TEST_CASE("CommandChain invariant enforcement", "[command_chain]")
{
	// This test manually tries to create invalid states to verify they're prevented

	// Test manual construction of a command chain with invalid operator count
	// This should not be possible with public API, but we'll verify with reflection if possible

	// We can test if the invariant check in parseFromTokens works by creating a malformed
	// token sequence that would violate the invariant if not checked

	// Example: command | command | (missing final command)
	std::vector<std::string> malformedTokens = {"ls", "|", "grep", "pattern", "|"};

	// The parse function should handle this by not adding the final pipe operator
	// without a command after it, maintaining the invariant
	CommandChain chain = CommandChain::parseFromTokens(malformedTokens);
	CHECK(chain.size() == 2);
	CHECK(chain.operators().size() == 1);

	// Verify invariant: operators.size() == commands.size() - 1
	CHECK(chain.operators().size() == chain.commands().size() - 1);

	// Try another malformed case: command && command || (missing final command)
	std::vector<std::string> malformedTokens2 = {"ls", "&&", "grep", "pattern", "||"};
	CommandChain chain2 = CommandChain::parseFromTokens(malformedTokens2);
	CHECK(chain2.size() == 2);
	CHECK(chain2.operators().size() == 1);

	// Verify invariant: operators.size() == commands.size() - 1
	CHECK(chain2.operators().size() == chain2.commands().size() - 1);
}
