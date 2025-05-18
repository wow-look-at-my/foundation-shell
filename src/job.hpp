#pragma once
static_assert(false, "File naming convention: This file should be renamed to Job.hpp");

#include <string>
#include <vector>
#include <sys/types.h>
#include <unistd.h>

// Job structure for background processes
struct Job {
    pid_t pid;
    std::string command;
    bool running;
    int jobId;
};

// Vector to store active jobs
extern std::vector<Job> activeJobs;

// Function to get the jobs file path
std::string getJobsFilePath();

// Function to add a job to the active jobs list
int addJob(pid_t pid, const std::string& command);

// Function to list all active jobs
void listJobs();

// Function to bring a background job to the foreground
int foregroundJob(int jobId);

// Function to continue a stopped job in the background
bool backgroundJob(int jobId);

// Signal handler for SIGCHLD (child process status change)
void sigchldHandler(int signo);
