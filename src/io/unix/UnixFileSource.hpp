#pragma once

#include "FDSource.hpp"
#include <filesystem>

// File-based implementation of ISource that wraps an FDSource
class FileSource final : public FDSource
{
public:
    // Open a file for reading using filesystem path
    explicit FileSource(const std::filesystem::path& path);
};