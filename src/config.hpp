#pragma once
static_assert(false, "File naming convention: This file should be renamed to Config.hpp");

#include <string>
#include <map>
#include <fstream>
#include <iostream>

// Shell configuration class
class ShellConfig {
public:
    ShellConfig();

    // Load configuration from file
    void loadConfig();

    // Save configuration to file
    void saveConfig();

    // Get a color from the current theme
    std::string getThemeColor(const std::string& key) const;

    // Format the prompt template
    std::string formatPrompt(const std::string& templ, int exitStatus) const;

    // Public properties
    std::string themeName;
    std::map<std::string, std::string> themeColors;
    std::string promptTemplate;
    bool showExitStatus;
    std::string welcomeMessage;
    bool colorEnabled;
    bool enableCommandSuggestions;
    int suggestionThreshold;

private:
    // Helper to get the configuration file path
    std::string getConfigFilePath() const;
};

// Global color constants
namespace Colors {
    extern const std::string COLOR_RESET;
    extern const std::string COLOR_RED;
    extern const std::string COLOR_GREEN;
    extern const std::string COLOR_YELLOW;
    extern const std::string COLOR_BLUE;
    extern const std::string COLOR_MAGENTA;
    extern const std::string COLOR_CYAN;
    extern const std::string COLOR_WHITE;
    extern const std::string COLOR_BOLD;
}

// Global constants
namespace Constants {
    extern const std::string HISTORY_FILE_NAME;
    extern const std::string JOBS_FILE_NAME;
    extern const std::string CONFIG_FILE_NAME;
    extern const std::string ALIASES_FILE_NAME;
    extern const int MAX_HISTORY_LINES;

    // Map of theme names to color values
    extern std::map<std::string, std::map<std::string, std::string>> predefinedThemes;
}

// Debug mode flag
extern bool debugMode;

// Global debug info structure
struct DebugInfo {
    std::chrono::steady_clock::time_point startTime;
    int commandCount = 0;
    int pipelineCount = 0;
    int redirectionCount = 0;
    int backgroundProcessCount = 0;
};

extern DebugInfo debugInfo;
