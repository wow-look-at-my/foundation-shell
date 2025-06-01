#!/bin/bash
# Script to detect and fix .find(...) == npos / != npos patterns in test files.
# Converts simple cases to exact equality or adds FAIL messages for complex cases.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
	local color=$1
	shift
	echo -e "${color}$*${NC}"
}

# Function to process a single file
process_file() {
	local file="$1"
	local dry_run="$2"
	local temp_file=$(mktemp)
	local modified=false
	local line_num=0

	print_color $YELLOW "Processing: $file"

	# Read file line by line
	while IFS= read -r line || [[ -n "$line" ]]; do
		((line_num++))
		local new_line="$line"
		local add_fail=false
		local add_fixme=false
		local suggestion=""

		# Check for .find(...) == std::string::npos pattern
		if echo "$line" | grep -q '\.find([^)]*)\s*==\s*std::string::npos'; then
			print_color $RED "$file:$line_num: Found .find() == npos pattern"

			# Try to extract simple quoted string patterns
			if echo "$line" | grep -q '\.find("[^"]*")\s*==\s*std::string::npos'; then
				# Extract the variable and search term
				var_name=$(echo "$line" | sed -n 's/.*\b\([a-zA-Z_][a-zA-Z0-9_]*\)\.find.*/\1/p')
				search_term=$(echo "$line" | sed -n 's/.*\.find("\([^"]*\)").*/\1/p')

				if [[ -n "$var_name" && -n "$search_term" ]]; then
					suggestion="CHECK($var_name != \"$search_term\");"
					add_fixme=true
					print_color $GREEN "  Suggestion: $suggestion"
				else
					add_fail=true
				fi
			else
				add_fail=true
			fi

		# Check for .find(...) != std::string::npos pattern
		elif echo "$line" | grep -q '\.find([^)]*)\s*!=\s*std::string::npos'; then
			print_color $RED "$file:$line_num: Found .find() != npos pattern"

			# Try to extract simple quoted string patterns
			if echo "$line" | grep -q '\.find("[^"]*")\s*!=\s*std::string::npos'; then
				var_name=$(echo "$line" | sed -n 's/.*\b\([a-zA-Z_][a-zA-Z0-9_]*\)\.find.*/\1/p')
				search_term=$(echo "$line" | sed -n 's/.*\.find("\([^"]*\)").*/\1/p')

				if [[ -n "$var_name" && -n "$search_term" ]]; then
					suggestion="CHECK($var_name == \"FULL_EXPECTED_STRING_HERE\"); // was checking for: $search_term"
					add_fixme=true
					print_color $YELLOW "  Needs manual fix: replace with exact equality check"
				else
					add_fail=true
				fi
			else
				add_fail=true
			fi

		# Check for .find( and npos in the same line (simpler check)
		elif echo "$line" | grep -q '\.find(' && echo "$line" | grep -q 'npos'; then
			print_color $RED "$file:$line_num: Found .find() and npos in same line"
			add_fail=true
		fi

		# Add appropriate fixes
		if [[ "$add_fail" == true ]]; then
			# Get indentation from original line
			local indent=$(echo "$line" | sed 's/[^ \t].*//')
			echo "${indent}FAIL(\"Use exact equality or die\");" >> "$temp_file"
			modified=true
			print_color $RED "  Added FAIL() statement"
		fi

		if [[ "$add_fixme" == true ]]; then
			# Get indentation from original line
			local indent=$(echo "$line" | sed 's/[^ \t].*//')
			echo "${indent}// FIXME: Convert to exact equality: $suggestion" >> "$temp_file"
			echo "${indent}// OLD: $(echo "$line" | sed 's/^[ \t]*//')" >> "$temp_file"
			modified=true
			print_color $GREEN "  Added FIXME comment with suggestion"
			# Skip the original line
			continue
		fi

		# Add the original line
		echo "$line" >> "$temp_file"

	done < "$file"

	# Replace original file if modified (unless dry-run)
	if [[ "$modified" == true ]]; then
		if [[ "$dry_run" == true ]]; then
			print_color $YELLOW "DRY-RUN: Would modify: $file"
			rm "$temp_file"
		else
			mv "$temp_file" "$file"
			print_color $GREEN "Modified: $file"
		fi
		return 0
	else
		rm "$temp_file"
		return 1
	fi
}

# Function to find test files
find_test_files() {
	local directory="$1"

	# Find .cpp and .hpp files in our test directories, excluding build and dependencies
	find "$directory" -type f \( -name "*.cpp" -o -name "*.hpp" \) \
		-path "*/tests/*" \
		! -path "*/build/*" \
		! -path "*/_deps/*" \
		! -path "*/extern/*" | \
	sort
}

# Main function
main() {
	local dry_run=false
	local directory="."

	# Parse arguments
	while [[ $# -gt 0 ]]; do
		case $1 in
			--dry-run)
				dry_run=true
				shift
				;;
			-*)
				echo "Unknown option $1"
				echo "Usage: $0 [--dry-run] [directory]"
				exit 1
				;;
			*)
				directory="$1"
				shift
				;;
		esac
	done

	if [[ "$dry_run" == true ]]; then
		print_color $YELLOW "DRY-RUN MODE: No files will be modified"
	fi

	print_color $YELLOW "Scanning for .find() patterns in test files under: $directory"

	# Find test files
	local test_files_list=$(find_test_files "$directory")

	if [[ -z "$test_files_list" ]]; then
		print_color $RED "No test files found!"
		exit 1
	fi

	local test_files_count=$(echo "$test_files_list" | wc -l)
	print_color $YELLOW "Found $test_files_count test files to process:"
	echo "$test_files_list" | sed 's/^/  /'
	echo

	local modified_count=0

	# Process each file
	echo "$test_files_list" | while IFS= read -r file; do
		if [[ -n "$file" ]]; then
			if process_file "$file" "$dry_run"; then
				((modified_count++))
			fi
		fi
	done

	echo
	print_color $YELLOW "Processed $test_files_count files, modified $modified_count files"

	if [[ $modified_count -gt 0 ]]; then
		echo
		print_color $GREEN "Next steps:"
		echo "1. Review the FIXME comments and convert to exact equality"
		echo "2. Fix any FAIL() statements by using proper exact equality"
		echo "3. Remove the comments once fixed"
		echo "4. Build and test to ensure everything works"
	fi
}

# Run main function with all arguments
main "$@"
