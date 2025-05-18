#pragma once

#include "ISink.hpp"
#include "ISource.hpp"
#include <memory>

// Interface for pipe operations
class IPipe 
{
public:
    virtual ~IPipe() = default;
    
    // Get the read end of the pipe
    virtual std::shared_ptr<ISource> getSource() const = 0;
    
    // Get the write end of the pipe
    virtual std::shared_ptr<ISink> getSink() const = 0;
};