#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "../src/CommandChain.hpp"
#include "../src/Token.hpp"
#include "../src/Command.hpp"
#include <stdexcept>
#include <string>
#include <vector>

// Test creating a CommandChain with valid commands
TEST(CommandChainTest, ValidConstruction)
{
    // Test creating empty chain
    EXPECT_NO_THROW(CommandChain());
    CommandChain emptyChain;
    EXPECT_TRUE(emptyChain.empty());
    EXPECT_EQ(emptyChain.size(), 0);
    
    // Test creating chain with a single command
    Command cmd;
    cmd.args = {"ls", "-la"};
    EXPECT_NO_THROW(CommandChain(cmd));
    
    CommandChain singleCmdChain(cmd);
    EXPECT_FALSE(singleCmdChain.empty());
    EXPECT_EQ(singleCmdChain.size(), 1);
    EXPECT_EQ(singleCmdChain.commands().size(), 1);
    EXPECT_EQ(singleCmdChain.operators().size(), 0);
    EXPECT_EQ(singleCmdChain.getCommand(0).args[0], "ls");
    EXPECT_EQ(singleCmdChain.getCommand(0).args[1], "-la");
    
    // Test appending a command to the chain
    Command cmd2;
    cmd2.args = {"grep", "pattern"};
    EXPECT_NO_THROW(singleCmdChain.appendCommand(cmd2, TokenType::Pipe));
    
    EXPECT_EQ(singleCmdChain.size(), 2);
    EXPECT_EQ(singleCmdChain.commands().size(), 2);
    EXPECT_EQ(singleCmdChain.operators().size(), 1);
    EXPECT_EQ(singleCmdChain.getOperator(0), TokenType::Pipe);
    
    // Test appending more commands
    Command cmd3;
    cmd3.args = {"wc", "-l"};
    EXPECT_NO_THROW(singleCmdChain.appendCommand(cmd3, TokenType::And));
    
    EXPECT_EQ(singleCmdChain.size(), 3);
    EXPECT_EQ(singleCmdChain.commands().size(), 3);
    EXPECT_EQ(singleCmdChain.operators().size(), 2);
    EXPECT_EQ(singleCmdChain.getOperator(1), TokenType::And);
    
    // Verify invariant: operators.size() == commands.size() - 1
    EXPECT_EQ(singleCmdChain.operators().size(), singleCmdChain.commands().size() - 1);
}

// Test invalid CommandChain constructions
TEST(CommandChainTest, InvalidConstruction)
{
    // Test creating chain with an empty command
    Command emptyCmd;
    EXPECT_THROW({CommandChain chain(emptyCmd);}, std::invalid_argument);
    
    // Test appending to an empty chain
    CommandChain emptyChain;
    Command validCmd;
    validCmd.args = {"ls"};
    EXPECT_THROW(emptyChain.appendCommand(validCmd, TokenType::Pipe), std::logic_error);
    
    // Test appending an empty command
    CommandChain chain(validCmd);
    Command emptyCmd2;
    EXPECT_THROW(chain.appendCommand(emptyCmd2, TokenType::Pipe), std::invalid_argument);
    
    // Test out-of-bounds access
    EXPECT_THROW(chain.getCommand(1), std::out_of_range);
    EXPECT_THROW(chain.getOperator(0), std::out_of_range);
    
    // Test invalid token parsing
    std::vector<std::string> invalidTokens = {"", "|", ""};
    // This shouldn't throw but should create a valid chain with no commands
    // The empty strings get filtered out during parsing
    EXPECT_NO_THROW(CommandChain::parseFromTokens(invalidTokens));
    CommandChain invalidChain = CommandChain::parseFromTokens(invalidTokens);
    EXPECT_TRUE(invalidChain.empty());
}

// Test parsing from tokens
TEST(CommandChainTest, ParseFromTokens)
{
    // Test parsing a simple command
    std::vector<std::string> simpleTokens = {"ls", "-la"};
    CommandChain simpleChain = CommandChain::parseFromTokens(simpleTokens);
    EXPECT_EQ(simpleChain.size(), 1);
    EXPECT_EQ(simpleChain.commands()[0].args[0], "ls");
    EXPECT_EQ(simpleChain.commands()[0].args[1], "-la");
    
    // Test parsing a piped command
    std::vector<std::string> pipedTokens = {"ls", "-la", "|", "grep", "pattern"};
    CommandChain pipedChain = CommandChain::parseFromTokens(pipedTokens);
    EXPECT_EQ(pipedChain.size(), 2);
    EXPECT_EQ(pipedChain.operators()[0], TokenType::Pipe);
    EXPECT_EQ(pipedChain.commands()[0].args[0], "ls");
    EXPECT_EQ(pipedChain.commands()[1].args[0], "grep");
    
    // Test parsing AND operator
    std::vector<std::string> andTokens = {"mkdir", "test", "&&", "cd", "test"};
    CommandChain andChain = CommandChain::parseFromTokens(andTokens);
    EXPECT_EQ(andChain.size(), 2);
    EXPECT_EQ(andChain.operators()[0], TokenType::And);
    
    // Test parsing OR operator
    std::vector<std::string> orTokens = {"ls", "nonexistent", "||", "echo", "not found"};
    CommandChain orChain = CommandChain::parseFromTokens(orTokens);
    EXPECT_EQ(orChain.size(), 2);
    EXPECT_EQ(orChain.operators()[0], TokenType::Or);
    
    // Test parsing with input redirection
    std::vector<std::string> inRedirTokens = {"sort", "<", "input.txt"};
    CommandChain inRedirChain = CommandChain::parseFromTokens(inRedirTokens);
    EXPECT_EQ(inRedirChain.size(), 1);
    EXPECT_EQ(inRedirChain.commands()[0].inputFile, "input.txt");
    
    // Test parsing with output redirection
    std::vector<std::string> outRedirTokens = {"ls", "-la", ">", "output.txt"};
    CommandChain outRedirChain = CommandChain::parseFromTokens(outRedirTokens);
    EXPECT_EQ(outRedirChain.size(), 1);
    EXPECT_EQ(outRedirChain.commands()[0].outputFile, "output.txt");
    EXPECT_FALSE(outRedirChain.commands()[0].appendOutput);
    
    // Test parsing with append output redirection
    std::vector<std::string> appendRedirTokens = {"echo", "hello", ">>", "output.txt"};
    CommandChain appendRedirChain = CommandChain::parseFromTokens(appendRedirTokens);
    EXPECT_EQ(appendRedirChain.size(), 1);
    EXPECT_EQ(appendRedirChain.commands()[0].outputFile, "output.txt");
    EXPECT_TRUE(appendRedirChain.commands()[0].appendOutput);
    
    // Test parsing with error redirection
    std::vector<std::string> errRedirTokens = {"gcc", "file.c", "2>", "error.log"};
    CommandChain errRedirChain = CommandChain::parseFromTokens(errRedirTokens);
    EXPECT_EQ(errRedirChain.size(), 1);
    EXPECT_EQ(errRedirChain.commands()[0].errorFile, "error.log");
    EXPECT_FALSE(errRedirChain.commands()[0].appendError);
    
    // Test parsing with append error redirection
    std::vector<std::string> appendErrRedirTokens = {"gcc", "file.c", "2>>", "error.log"};
    CommandChain appendErrRedirChain = CommandChain::parseFromTokens(appendErrRedirTokens);
    EXPECT_EQ(appendErrRedirChain.size(), 1);
    EXPECT_EQ(appendErrRedirChain.commands()[0].errorFile, "error.log");
    EXPECT_TRUE(appendErrRedirChain.commands()[0].appendError);
    
    // Test parsing with background process
    std::vector<std::string> backgroundTokens = {"sleep", "10", "&"};
    CommandChain backgroundChain = CommandChain::parseFromTokens(backgroundTokens);
    EXPECT_EQ(backgroundChain.size(), 1);
    EXPECT_TRUE(backgroundChain.commands()[0].backgroundProcess);
    
    // Test parsing complex command chain
    std::vector<std::string> complexTokens = {
        "grep", "error", "<", "log.txt", "|", 
        "sort", "|", 
        "uniq", "-c", ">", "results.txt", "&&", 
        "echo", "done", "||", 
        "echo", "failed"
    };
    CommandChain complexChain = CommandChain::parseFromTokens(complexTokens);
    EXPECT_EQ(complexChain.size(), 5);
    EXPECT_EQ(complexChain.operators()[0], TokenType::Pipe);
    EXPECT_EQ(complexChain.operators()[1], TokenType::Pipe);
    EXPECT_EQ(complexChain.operators()[2], TokenType::And);
    EXPECT_EQ(complexChain.operators()[3], TokenType::Or);
    EXPECT_EQ(complexChain.commands()[0].inputFile, "log.txt");
    EXPECT_EQ(complexChain.commands()[2].outputFile, "results.txt");
    
    // Verify invariant: operators.size() == commands.size() - 1
    EXPECT_EQ(complexChain.operators().size(), complexChain.commands().size() - 1);
}

// Test edge cases
TEST(CommandChainTest, EdgeCases)
{
    // Test empty tokens
    std::vector<std::string> emptyTokens = {};
    CommandChain emptyChain = CommandChain::parseFromTokens(emptyTokens);
    EXPECT_TRUE(emptyChain.empty());
    
    // Test tokens with only operators (invalid command but shouldn't crash)
    std::vector<std::string> onlyOperators = {"|", "&&", "||"};
    CommandChain onlyOperatorsChain = CommandChain::parseFromTokens(onlyOperators);
    EXPECT_TRUE(onlyOperatorsChain.empty());
    
    // Test missing redirection targets
    std::vector<std::string> missingTarget = {"cat", ">"};
    CommandChain missingTargetChain = CommandChain::parseFromTokens(missingTarget);
    EXPECT_EQ(missingTargetChain.size(), 1);
    EXPECT_TRUE(missingTargetChain.commands()[0].outputFile.empty());
    
    // Test & not at the end
    std::vector<std::string> invalidBackground = {"ls", "&", "-la"};
    CommandChain invalidBackgroundChain = CommandChain::parseFromTokens(invalidBackground);
    EXPECT_EQ(invalidBackgroundChain.size(), 1);
    EXPECT_FALSE(invalidBackgroundChain.commands()[0].backgroundProcess);
    EXPECT_EQ(invalidBackgroundChain.commands()[0].args[1], "&");
}

// Test invariant enforcement - making sure it's impossible to create an invalid state
TEST(CommandChainTest, InvariantEnforcement)
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
    EXPECT_EQ(chain.size(), 2);
    EXPECT_EQ(chain.operators().size(), 1);
    
    // Verify invariant: operators.size() == commands.size() - 1
    EXPECT_EQ(chain.operators().size(), chain.commands().size() - 1);
    
    // Try another malformed case: command && command || (missing final command)
    std::vector<std::string> malformedTokens2 = {"ls", "&&", "grep", "pattern", "||"};
    CommandChain chain2 = CommandChain::parseFromTokens(malformedTokens2);
    EXPECT_EQ(chain2.size(), 2);
    EXPECT_EQ(chain2.operators().size(), 1);
    
    // Verify invariant: operators.size() == commands.size() - 1
    EXPECT_EQ(chain2.operators().size(), chain2.commands().size() - 1);
}

// Main function provided by gtest_main library