#pragma once

#include <memory>

#include "ISink.hpp"
#include "ISource.hpp"

/**
 * Interface for pipe operations
 */
class IPipe
{
public:
	virtual ~IPipe() = default;

	/**
	 * Create a platform-specific pipe implementation
	 * @return Shared pointer to IPipe interface
	 */
	static std::shared_ptr<IPipe> create();

	/**
	 * Get the read end of the pipe
	 * @return Shared pointer to ISource interface
	 */
	virtual std::shared_ptr<ISource> getSource() const = 0;

	/**
	 * Get the write end of the pipe
	 * @return Shared pointer to ISink interface
	 */
	virtual std::shared_ptr<ISink> getSink() const = 0;
};
