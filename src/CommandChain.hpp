#pragma once

#include "command.hpp"
#include "task.hpp"
#include "config.hpp"
#include "Token.hpp"
#include <vector>
#include <string>
#include <stdexcept>
#include <utility>

// Forward declaration
class CommandChain;

// A class representing a chain of commands connected by operators
class CommandChain 
{
public:
    // Default constructor - creates an empty chain
    CommandChain() = default;
    
    // Constructor that parses from tokens
    explicit CommandChain(const std::vector<std::string>& tokens);
    
    // Static method to parse tokens into a command chain
    static CommandChain parseFromTokens(const std::vector<std::string>& tokens);
    
    // Execute the command chain asynchronously
    Task<int> executeAsync(const ShellConfig& config) const;
    
    // Start a new chain with a single command
    explicit CommandChain(Command firstCommand) {
        if (firstCommand.args.empty()) {
            throw std::invalid_argument("Cannot add empty command to chain");
        }
        commands_.push_back(std::move(firstCommand));
    }
    
    // Add a command with its operator to the chain
    // This is the only way to add more commands after the first one
    void appendCommand(Command nextCommand, TokenType op) {
        if (commands_.empty()) {
            throw std::logic_error("Cannot append command to empty chain - use the constructor with a command first");
        }
        if (nextCommand.args.empty()) {
            throw std::invalid_argument("Cannot add empty command to chain");
        }
        
        operators_.push_back(op);
        commands_.push_back(std::move(nextCommand));
    }
    
    // Accessors
    const std::vector<Command>& commands() const { return commands_; }
    const std::vector<TokenType>& operators() const { return operators_; }
    const Command& getCommand(size_t index) const { 
        if (index >= commands_.size()) {
            throw std::out_of_range("Command index out of range");
        }
        return commands_[index];
    }
    TokenType getOperator(size_t index) const {
        if (index >= operators_.size()) {
            throw std::out_of_range("Operator index out of range");
        }
        return operators_[index];
    }
    
    // Check if empty
    bool empty() const { return commands_.empty(); }
    
    // Size (number of commands)
    size_t size() const { return commands_.size(); }
    

private:
    std::vector<Command> commands_;
    std::vector<TokenType> operators_; // operators_[i] is the operator between commands_[i] and commands_[i+1]
};