#pragma once

#include <cstdlib>
#include <string>
#include <utility>
#include <vector>

/**
 * RAII helper for temporarily setting environment variables
 * Automatically restores original values when destroyed
 */
class EnvManager
{
public:
	/**
	 * Default constructor
	 */
	EnvManager() = default;
	/**
	 * Set an environment variable temporarily
	 * @param name Variable name
	 * @param value Variable value
	 */
	void set(const std::string& name, const std::string& value)
	{
		const char* old_value = getenv(name.c_str());
		saved_values_.emplace_back(name, old_value ? old_value : "", old_value != nullptr);
		setenv(name.c_str(), value.c_str(), 1);
	}

	/**
	 * Set multiple environment variables from assignments
	 * @param env_assignments Vector of name-value pairs
	 */
	void set_assignments(const std::vector<std::pair<std::string, std::string>>& env_assignments)
	{
		for (const auto& [name, value] : env_assignments)
		{
			set(name, value);
		}
	}

	/**
	 * Destructor - automatically restores all environment variables
	 */
	~EnvManager()
	{
		for (const auto& [name, saved_value, was_set] : saved_values_)
		{
			if (was_set)
			{
				setenv(name.c_str(), saved_value.c_str(), 1);
			}
			else
			{
				unsetenv(name.c_str());
			}
		}
	}

	// Non-copyable, non-movable for simplicity
	EnvManager(const EnvManager&) = delete;
	EnvManager& operator=(const EnvManager&) = delete;
	EnvManager(EnvManager&&) = delete;
	EnvManager& operator=(EnvManager&&) = delete;

private:
	// Store name, original value, and whether it was originally set
	std::vector<std::tuple<std::string, std::string, bool>> saved_values_;
};