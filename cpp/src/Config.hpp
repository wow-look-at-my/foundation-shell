#pragma once

#include <string_view>

// Global color constants
namespace Colors
{
inline constexpr std::string_view COLOR_RESET = "\033[0m";
inline constexpr std::string_view COLOR_RED = "\033[31m";
inline constexpr std::string_view COLOR_GREEN = "\033[32m";
inline constexpr std::string_view COLOR_YELLOW = "\033[33m";
inline constexpr std::string_view COLOR_BLUE = "\033[34m";
inline constexpr std::string_view COLOR_MAGENTA = "\033[35m";
inline constexpr std::string_view COLOR_CYAN = "\033[36m";
inline constexpr std::string_view COLOR_WHITE = "\033[37m";
inline constexpr std::string_view COLOR_BOLD = "\033[1m";
} // namespace Colors

// Global constants
namespace Constants
{
inline constexpr int MAX_HISTORY_LINES = 1000;
}

// Error message constants
namespace ErrorMessages
{
inline constexpr std::string_view COMMAND_NOT_FOUND = "Failed to execute command: {}";
inline constexpr std::string_view FS_ENTRY_NOT_FOUND = "No such file or directory";
inline constexpr std::string_view FAILED_TO_FORK = "Failed to fork process";
inline constexpr std::string_view ERROR_WAITING_FOR_PROCESS = "Error waiting for process";
} // namespace ErrorMessages
