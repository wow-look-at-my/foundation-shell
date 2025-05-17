#include <iostream>
#include <string>
#include <vector>
#include <sstream>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>
#include <cstring>
#include <wordexp.h>
#include <fstream>
#include <filesystem>

// Function to split a string into tokens respecting quotes and escapes using wordexp
std::vector<std::string> bashSplitString(const std::string& input) {
	std::vector<std::string> tokens;
	wordexp_t p;

	// Use wordexp to perform bash-like word expansion
	int status = wordexp(input.c_str(), &p, 0);
	if (status == 0) {
		// Copy the expanded words to our vector
		for (size_t i = 0; i < p.we_wordc; i++) {
			tokens.push_back(p.we_wordv[i]);
		}
		wordfree(&p);
	} else {
		// If wordexp failed, fallback to simple splitting
		std::istringstream iss(input);
		std::string token;
		while (iss >> token) {
			tokens.push_back(token);
		}
	}

	return tokens;
}

// Function to convert vector of strings to array of C-strings
char** vectorToCharArray(const std::vector<std::string>& args) {
	char** result = new char*[args.size() + 1]; // +1 for the NULL terminator

	for (size_t i = 0; i < args.size(); i++) {
		result[i] = new char[args[i].size() + 1];
		std::strcpy(result[i], args[i].c_str());
	}

	result[args.size()] = nullptr; // Null-terminate the array
	return result;
}

// Function to free memory allocated for char array
void freeCharArray(char** array, int size) {
	for (int i = 0; i < size; i++) {
		delete[] array[i];
	}
	delete[] array;
}

// Function to execute a command
int executeCommand(const std::vector<std::string>& args) {
	if (args.empty()) {
		return 0;
	}

	// Handle built-in commands
	if (args[0] == "cd") {
		if (args.size() < 2) {
			// If no path is provided, change to HOME directory
			const char* homeDir = getenv("HOME");
			if (homeDir == nullptr) {
				std::cerr << "HOME environment variable not set\n";
				return 1;
			}
			if (chdir(homeDir) != 0) {
				std::cerr << "Failed to change directory to " << homeDir << "\n";
				return 1;
			}
		} else {
			if (chdir(args[1].c_str()) != 0) {
				std::cerr << "Failed to change directory to " << args[1] << "\n";
				return 1;
			}
		}
		return 0;
	}

	if (args[0] == "exit") {
		// Exit the shell
		exit(0);
	}

	// Fork a child process
	pid_t pid = fork();

	if (pid == -1) {
		// Fork failed
		std::cerr << "Fork failed\n";
		return 1;
	} else if (pid == 0) {
		// Child process
		char** argArray = vectorToCharArray(args);

		// Execute the command
		execvp(argArray[0], argArray);

		// If execvp returns, an error occurred
		std::cerr << "Command not found: " << args[0] << "\n";
		freeCharArray(argArray, args.size());
		exit(1);
	} else {
		// Parent process
		int status;
		waitpid(pid, &status, 0);
		return WEXITSTATUS(status);
	}
}

int main() {
	std::string input;
	std::string prompt = "base-shell$ ";

	while (true) {
		// Print the prompt
		std::cout << prompt;

		// Get user input
		if (!std::getline(std::cin, input)) {
			// Handle EOF (Ctrl+D)
			std::cout << "\nExiting shell\n";
			break;
		}

		// Split the input into command and arguments, using bash-compatible splitting
		std::vector<std::string> args = bashSplitString(input);

		// Execute the command
		executeCommand(args);
	}

	return 0;
}
