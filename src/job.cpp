#include "job.hpp"
#include "config.hpp"
#include <iostream>
#include <algorithm>
#include <signal.h>
#include <sys/wait.h>
#include <fstream>
#include <cstdlib>

// Initialize active jobs vector
std::vector<Job> activeJobs;

// Function to get the jobs file path
std::string getJobsFilePath() {
    const char* homeDir = getenv("HOME");
    if (!homeDir) {
        return Constants::JOBS_FILE_NAME; // Fallback to current directory
    }
    return std::string(homeDir) + "/" + Constants::JOBS_FILE_NAME;
}

// Function to add a job to the active jobs list
int addJob(pid_t pid, const std::string& command) {
    // Clean up completed jobs first
    for (auto it = activeJobs.begin(); it != activeJobs.end();) {
        int status;
        pid_t result = waitpid(it->pid, &status, WNOHANG);
        if (result > 0) {
            // Process has ended
            it = activeJobs.erase(it);
        } else {
            ++it;
        }
    }

    // Assign a new job ID
    int newJobId = 1;
    if (!activeJobs.empty()) {
        newJobId = activeJobs.back().jobId + 1;
    }

    // Add the new job
    Job newJob = {pid, command, true, newJobId};
    activeJobs.push_back(newJob);

    return newJobId;
}

// Function to list all active jobs
void listJobs() {
    for (auto it = activeJobs.begin(); it != activeJobs.end();) {
        int status;
        pid_t result = waitpid(it->pid, &status, WNOHANG);

        if (result > 0) {
            // Process has ended
            std::cout << "[" << it->jobId << "] " << Colors::COLOR_RED << "Done" << Colors::COLOR_RESET << "                " << it->command << std::endl;
            it = activeJobs.erase(it);
        } else {
            // Process is still running
            std::cout << "[" << it->jobId << "] " << Colors::COLOR_GREEN << "Running" << Colors::COLOR_RESET << "             " << it->command << std::endl;
            ++it;
        }
    }
}

// Function to bring a background job to the foreground
int foregroundJob(int jobId) {
    for (const auto& job : activeJobs) {
        if (job.jobId == jobId) {
            int status;
            std::cout << job.command << std::endl;

            // Wait for the process to complete
            waitpid(job.pid, &status, 0);

            // Remove the job from the active jobs list
            activeJobs.erase(
                std::remove_if(activeJobs.begin(), activeJobs.end(),
                              [jobId](const Job& j) { return j.jobId == jobId; }),
                activeJobs.end());

            return WEXITSTATUS(status);
        }
    }

    std::cerr << "Job " << jobId << " not found" << std::endl;
    return 1;
}

// Function to continue a stopped job in the background
bool backgroundJob(int jobId) {
    for (auto& job : activeJobs) {
        if (job.jobId == jobId) {
            if (kill(job.pid, SIGCONT) == 0) {
                std::cout << "[" << job.jobId << "] " << job.command << " &" << std::endl;
                job.running = true;
                return true;
            } else {
                std::cerr << "Failed to continue job " << jobId << std::endl;
                return false;
            }
        }
    }

    std::cerr << "Job " << jobId << " not found" << std::endl;
    return false;
}

// Signal handler for SIGCHLD (child process status change)
void sigchldHandler(int signo) {
    // Clean up zombie processes
    while (waitpid(-1, nullptr, WNOHANG) > 0);
}
