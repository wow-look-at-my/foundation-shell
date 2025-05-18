#pragma once

#include "ISink.hpp"
#include <filesystem>
#include <memory>

// Platform-specific file sink implementation
#if defined(_WIN32) || defined(_WIN64)
  // Windows implementation would go here in the future
  #error "Windows implementation not yet available"
#else
  // Unix implementation
  #include "unix/UnixFileSink.hpp"
  using FileSink = UnixFileSink;
#endif

// Typedef for shared pointer to FileSink
using FileSinkPtr = std::shared_ptr<FileSink>;