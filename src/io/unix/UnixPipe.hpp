#pragma once

#include "IPipe.hpp"
#include "FDSource.hpp"
#include "FDSink.hpp"
#include <memory>

// Unix pipe implementation using file descriptors
class UnixPipe final : public IPipe 
{
public:
    // Create a new pipe
    UnixPipe();
    
    // Get the read end of the pipe
    std::shared_ptr<ISource> getSource() const override;
    
    // Get the write end of the pipe
    std::shared_ptr<ISink> getSink() const override;
    
private:
    std::shared_ptr<FDSource> source;
    std::shared_ptr<FDSink> sink;
};