#pragma once

#include <string>
#include <vector>
#include <unistd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <fcntl.h>

// Structure to represent a command with its I/O redirections
class Command {
public:
    Command();
    ~Command() = default;

    // Command arguments
    std::vector<std::string> args;

    // I/O redirection files
    std::string inputFile;
    std::string outputFile;
    std::string errorFile;

    // Redirection modes
    bool appendOutput = false;
    bool appendError = false;
    bool backgroundProcess = false;

    // Execute the command with optional I/O redirection
    int execute(int inputFd = STDIN_FILENO, int outputFd = STDOUT_FILENO) const;

private:
    // Helper method to handle built-in commands
    int handleBuiltins() const;

    // Common I/O setup for child process
    void setupChildIO(int inputFd, int outputFd) const;

    // Convert vector of strings to array of C-strings
    char** vectorToCharArray(const std::vector<std::string>& args) const;

    // Free memory allocated for char array
    void freeCharArray(char** array, int size) const;
};

// Function to parse input into commands with redirections and pipes
std::vector<Command> parseCommand(const std::vector<std::string>& tokens);

// Function to execute a pipeline of commands
int executePipeline(const std::vector<Command>& commands);

// Function to split a string into tokens respecting quotes and escapes
std::vector<std::string> bashSplitString(const std::string& input);
