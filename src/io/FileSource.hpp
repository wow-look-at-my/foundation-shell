#pragma once

#include "ISource.hpp"
#include <filesystem>
#include <memory>

// Platform-specific file source implementation
#if defined(_WIN32) || defined(_WIN64)
  // Windows implementation would go here in the future
  #error "Windows implementation not yet available"
#else
  // Unix implementation
  #include "unix/UnixFileSource.hpp"
  using FileSource = UnixFileSource;
#endif

// Typedef for shared pointer to FileSource
using FileSourcePtr = std::shared_ptr<FileSource>;