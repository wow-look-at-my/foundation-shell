#pragma once

#include "FDSource.hpp"
#include <filesystem>

/**
 * Unix-specific implementation of ISource that reads from a file using file descriptors
 */
class UnixFileSource final : public FDSource
{
public:
    /**
     * Constructor that opens a file for reading
     * @param path Path to the file to open
     * @throws std::runtime_error if file cannot be opened
     */
    explicit UnixFileSource(const std::filesystem::path& path);
};