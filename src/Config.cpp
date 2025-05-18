#include "Config.hpp"
#include <unistd.h>
#include <limits.h>
#include <cstdlib>
#include <algorithm>

// Define color constants
namespace Colors
{
	const std::string COLOR_RESET = "\033[0m";
	const std::string COLOR_RED = "\033[31m";
	const std::string COLOR_GREEN = "\033[32m";
	const std::string COLOR_YELLOW = "\033[33m";
	const std::string COLOR_BLUE = "\033[34m";
	const std::string COLOR_MAGENTA = "\033[35m";
	const std::string COLOR_CYAN = "\033[36m";
	const std::string COLOR_WHITE = "\033[37m";
	const std::string COLOR_BOLD = "\033[1m";
}

// Define global constants
namespace Constants
{
	const std::string HISTORY_FILE_NAME = ".foundation_shell_history";
	const std::string JOBS_FILE_NAME = ".foundation_shell_jobs";
	const std::string CONFIG_FILE_NAME = ".foundation_shell_config";
	const std::string ALIASES_FILE_NAME = ".foundation_shell_aliases";
	const int MAX_HISTORY_LINES = 1000;

	// Initialize predefined themes
	std::map<std::string, std::map<std::string, std::string>> predefinedThemes = {
		{"default", {{"directory", Colors::COLOR_BLUE}, {"prompt", Colors::COLOR_RESET}, {"success", Colors::COLOR_GREEN}, {"error", Colors::COLOR_RED}, {"command", Colors::COLOR_WHITE}, {"header", Colors::COLOR_CYAN + Colors::COLOR_BOLD}}},
		{"dark", {{"directory", Colors::COLOR_CYAN}, {"prompt", Colors::COLOR_YELLOW}, {"success", Colors::COLOR_GREEN}, {"error", Colors::COLOR_RED}, {"command", Colors::COLOR_MAGENTA}, {"header", Colors::COLOR_YELLOW + Colors::COLOR_BOLD}}},
		{"light", {{"directory", Colors::COLOR_BLUE}, {"prompt", Colors::COLOR_MAGENTA}, {"success", Colors::COLOR_GREEN}, {"error", Colors::COLOR_RED}, {"command", Colors::COLOR_BLUE}, {"header", Colors::COLOR_BLUE + Colors::COLOR_BOLD}}},
		{"monochrome", {{"directory", Colors::COLOR_WHITE}, {"prompt", Colors::COLOR_WHITE}, {"success", Colors::COLOR_WHITE}, {"error", Colors::COLOR_WHITE}, {"command", Colors::COLOR_WHITE}, {"header", Colors::COLOR_WHITE + Colors::COLOR_BOLD}}}};
}

// Initialize global variables
bool debugMode = false;
DebugInfo debugInfo;

// ShellConfig implementation
ShellConfig::ShellConfig()
	: themeName("default"),
	  promptTemplate("{{status}} {{dir}} $ "),
	  showExitStatus(true),
	  welcomeMessage("Welcome to Foundation Shell"),
	  colorEnabled(true),
	  enableCommandSuggestions(true),
	  suggestionThreshold(3)
{
	// Initialize theme colors with default theme
	themeColors = Constants::predefinedThemes[themeName];
}

std::string ShellConfig::getConfigFilePath() const
{
	const char *homeDir = getenv("HOME");
	if (!homeDir)
	{
		return Constants::CONFIG_FILE_NAME; // Fallback to current directory
	}
	return std::string(homeDir) + "/" + Constants::CONFIG_FILE_NAME;
}

void ShellConfig::loadConfig()
{
	std::string configPath = getConfigFilePath();
	std::ifstream configFile(configPath);

	// If config file doesn't exist, use defaults
	if (!configFile.is_open())
	{
		// Set default theme colors
		themeColors = Constants::predefinedThemes[themeName];
		return;
	}

	std::string line;
	while (std::getline(configFile, line))
	{
		// Skip empty lines and comments
		if (line.empty() || line[0] == '#')
		{
			continue;
		}

		// Parse key-value pairs
		size_t equalsPos = line.find('=');
		if (equalsPos != std::string::npos)
		{
			std::string key = line.substr(0, equalsPos);
			std::string value = line.substr(equalsPos + 1);

			// Remove whitespace
			key.erase(0, key.find_first_not_of(" \t"));
			key.erase(key.find_last_not_of(" \t") + 1);
			value.erase(0, value.find_first_not_of(" \t"));
			value.erase(value.find_last_not_of(" \t") + 1);

			if (key == "theme")
			{
				themeName = value;
				// Check if it's a predefined theme
				if (Constants::predefinedThemes.find(value) != Constants::predefinedThemes.end())
				{
					themeColors = Constants::predefinedThemes[value];
				}
			}
			else if (key == "prompt_template")
			{
				promptTemplate = value;
			}
			else if (key == "show_exit_status")
			{
				showExitStatus = (value == "true" || value == "1" || value == "yes");
			}
			else if (key == "welcome_message")
			{
				welcomeMessage = value;
			}
			else if (key == "color_enabled")
			{
				colorEnabled = (value == "true" || value == "1" || value == "yes");
			}
			else if (key.substr(0, 6) == "color_")
			{
				// Custom color definition
				std::string colorKey = key.substr(6);
				themeColors[colorKey] = value;
			}
		}
	}

	configFile.close();
}

void ShellConfig::saveConfig()
{
	std::string configPath = getConfigFilePath();
	std::ofstream configFile(configPath);

	if (configFile.is_open())
	{
		configFile << "# Foundation Shell Configuration File\n\n";
		configFile << "theme=" << themeName << "\n";
		configFile << "prompt_template=" << promptTemplate << "\n";
		configFile << "show_exit_status=" << (showExitStatus ? "true" : "false") << "\n";
		configFile << "welcome_message=" << welcomeMessage << "\n";
		configFile << "color_enabled=" << (colorEnabled ? "true" : "false") << "\n";

		// Only save custom colors that differ from the predefined theme
		if (Constants::predefinedThemes.find(themeName) != Constants::predefinedThemes.end())
		{
			auto &defaultColors = Constants::predefinedThemes[themeName];
			for (const auto &[key, value] : themeColors)
			{
				if (defaultColors.find(key) == defaultColors.end() || defaultColors.at(key) != value)
				{
					configFile << "color_" << key << "=" << value << "\n";
				}
			}
		}
		else
		{
			// Save all colors for custom themes
			for (const auto &[key, value] : themeColors)
			{
				configFile << "color_" << key << "=" << value << "\n";
			}
		}

		configFile.close();
	}
}

std::string ShellConfig::getThemeColor(const std::string &key) const
{
	if (!colorEnabled)
	{
		return "";
	}

	auto it = themeColors.find(key);
	if (it != themeColors.end())
	{
		return it->second;
	}
	return "";
}

std::string ShellConfig::formatPrompt(const std::string &templ, int exitStatus) const
{
	std::string result = templ;

	// Get current directory
	char cwd[PATH_MAX];
	std::string dirStr = "unknown";
	if (getcwd(cwd, sizeof(cwd)) != nullptr)
	{
		dirStr = cwd;
	}

	// Replace {{dir}} with the current directory
	size_t dirPos = result.find("{{dir}}");
	if (dirPos != std::string::npos)
	{
		result.replace(dirPos, 7, getThemeColor("directory") + dirStr + Colors::COLOR_RESET);
	}

	// Replace {{status}} with exit status indicator
	size_t statusPos = result.find("{{status}}");
	if (statusPos != std::string::npos)
	{
		std::string statusIndicator;
		if (showExitStatus)
		{
			statusIndicator = exitStatus == 0 ? getThemeColor("success") + "✓" : getThemeColor("error") + "✗";
			statusIndicator += Colors::COLOR_RESET;
		}
		result.replace(statusPos, 10, statusIndicator);
	}

	return result;
}
