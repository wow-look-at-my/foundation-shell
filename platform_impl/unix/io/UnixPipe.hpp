#pragma once

#include "src/io/IPipe.hpp"
#include "FDSource.hpp"
#include "FDSink.hpp"
#include <memory>

/**
 * Unix-specific implementation of the IPipe interface
 * Creates a pipe using the underlying POSIX pipe() function
 */
class UnixPipe : public IPipe
{
public:
	/**
	 * Constructor that creates a new pipe
	 * @throws std::runtime_error if pipe creation fails
	 */
	UnixPipe();

	/**
	 * Get the read end of the pipe
	 * @return Shared pointer to the source interface
	 */
	std::shared_ptr<ISource> getSource() const override;

	/**
	 * Get the write end of the pipe
	 * @return Shared pointer to the sink interface
	 */
	std::shared_ptr<ISink> getSink() const override;

private:
	std::shared_ptr<FDSource> source; // Read end of the pipe
	std::shared_ptr<FDSink> sink;	  // Write end of the pipe
};
