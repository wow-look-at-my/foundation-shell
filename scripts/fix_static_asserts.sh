#!/bin/bash
# Script to:
# 1. Convert static_assert(true, ...) to static_assert(false, ...)
# 2. Convert // TODO: comments to static_assert(false, "TODO: ...") statements

echo "Checking for static_assert patterns and TODO comments..."

# Find all C++ files in the source directory
cpp_files=$(find ./src -type f -name "*.cpp" -o -name "*.hpp")

# Counter for modified files
modified_count=0
todo_count=0

for file in $cpp_files; do
    file_modified=0
    
    # Check if the file contains the incorrect static_assert pattern
    if grep -q "static_assert(true," "$file"; then
        echo "Found incorrect static_assert in $file"
        
        # Create a backup of the file if not already done
        if [ $file_modified -eq 0 ]; then
            cp "$file" "${file}.bak"
            file_modified=1
        fi
        
        # Replace all occurrences of static_assert(true, with static_assert(false,
        sed -i '' 's/static_assert(true,/static_assert(false,/g' "$file"
        
        # Increment the counter
        ((modified_count++))
    fi
    
    # Check for TODOs in comments that should be static assertions
    # Look for // TODO: patterns that are not already followed by static_assert
    if grep -q "// TODO:" "$file" && ! grep -q "// TODO:.*static_assert" "$file"; then
        # Create a backup of the file if not already done
        if [ $file_modified -eq 0 ]; then
            cp "$file" "${file}.bak"
            file_modified=1
        fi
        
        echo "Found TODO comments in $file"
        
        # Create a temporary file to process the replacements
        temp_file=$(mktemp)
        
        # Process the file line by line
        line_num=1
        while IFS= read -r line; do
            if echo "$line" | grep -q "// TODO:"; then
                # Extract the TODO message
                todo_message=$(echo "$line" | sed -E 's/.*\/\/ TODO:(.*)/\1/' | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
                
                # Get indentation of the original line
                indentation=$(echo "$line" | sed -E 's/^([[:space:]]*).*$/\1/')
                
                # Write the original comment line
                echo "$line" >> "$temp_file"
                
                # Add the static_assert line with proper indentation
                echo "${indentation}static_assert(false, \"TODO:$todo_message\");" >> "$temp_file"
                
                ((todo_count++))
            else
                # Pass through unchanged
                echo "$line" >> "$temp_file"
            fi
            ((line_num++))
        done < "$file"
        
        # Replace the original file with the modified content
        mv "$temp_file" "$file"
    fi
    
    # Mark the file as modified for the counter
    if [ $file_modified -eq 1 ]; then
        ((modified_count++))
    fi
done

if [ $modified_count -eq 0 ]; then
    echo "No issues found."
else
    echo "Fixed $modified_count file(s):"
    echo "- Fixed incorrect static_assert patterns"
    echo "- Converted $todo_count TODO comments to static_assert statements"
fi

exit 0