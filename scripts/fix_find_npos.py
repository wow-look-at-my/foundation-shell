#!/usr/bin/env python3
"""
Script to detect and fix .find(...) == npos / != npos patterns in test files.
Converts simple cases to exact equality or adds FAIL messages for complex cases.
"""

import re
import os
import sys
from pathlib import Path

def analyze_find_pattern(line, line_num, file_lines):
    """Analyze a line containing .find() and suggest fixes."""
    
    # Pattern for .find(...) == std::string::npos
    npos_equal_pattern = r'(\w+)\.find\([^)]+\)\s*==\s*std::string::npos'
    # Pattern for .find(...) != std::string::npos  
    npos_not_equal_pattern = r'(\w+)\.find\([^)]+\)\s*!=\s*std::string::npos'
    
    # Check for exact patterns we can fix
    npos_equal_match = re.search(npos_equal_pattern, line)
    npos_not_equal_match = re.search(npos_not_equal_pattern, line)
    
    if npos_equal_match or npos_not_equal_match:
        # Try to extract the search term for simple cases
        find_content_pattern = r'\.find\("([^"]+)"\)'
        find_match = re.search(find_content_pattern, line)
        
        if find_match:
            search_term = find_match.group(1)
            var_name = npos_equal_match.group(1) if npos_equal_match else npos_not_equal_match.group(1)
            
            if npos_equal_match:
                # .find("term") == npos means string doesn't contain term
                # Convert to: var != "term" (if it's a simple case)
                return f"SUGGESTION: {var_name} != \"{search_term}\""
            else:
                # .find("term") != npos means string contains term  
                # Convert to: var == "expected_full_string" (needs manual intervention)
                return f"NEEDS_MANUAL: {var_name} == \"FULL_EXPECTED_STRING_HERE\""
        else:
            return "COMPLEX_FIND_PATTERN"
    
    # Check for find and npos in close proximity (within 3 lines)
    if '.find(' in line or 'npos' in line:
        context_lines = file_lines[max(0, line_num-2):line_num+3]
        context_text = ''.join(context_lines)
        
        if '.find(' in context_text and 'npos' in context_text:
            return "PROXIMITY_MATCH"
    
    return None

def process_file(file_path):
    """Process a single file to detect and fix .find() patterns."""
    
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            lines = f.readlines()
    except Exception as e:
        print(f"Error reading {file_path}: {e}")
        return False
    
    modified = False
    new_lines = []
    
    for i, line in enumerate(lines):
        analysis = analyze_find_pattern(line, i, lines)
        
        if analysis:
            print(f"{file_path}:{i+1}: {analysis}")
            
            if analysis == "COMPLEX_FIND_PATTERN" or analysis == "PROXIMITY_MATCH":
                # Insert FAIL message before the problematic line
                indent = len(line) - len(line.lstrip())
                fail_line = ' ' * indent + 'FAIL("Use exact equality or die");\n'
                new_lines.append(fail_line)
                modified = True
            elif analysis.startswith("SUGGESTION:"):
                # For simple cases, comment out the old line and suggest new one
                suggestion = analysis.replace("SUGGESTION: ", "")
                indent = len(line) - len(line.lstrip())
                new_lines.append(' ' * indent + f'// FIXME: Convert to exact equality: CHECK({suggestion});\n')
                new_lines.append(' ' * indent + f'// OLD: {line.strip()}\n')
                modified = True
                continue  # Skip adding the original line
            elif analysis.startswith("NEEDS_MANUAL:"):
                # Add comment suggesting manual fix
                suggestion = analysis.replace("NEEDS_MANUAL: ", "")
                indent = len(line) - len(line.lstrip())
                new_lines.append(' ' * indent + f'// FIXME: Replace with exact equality: CHECK({suggestion});\n')
                modified = True
        
        new_lines.append(line)
    
    if modified:
        try:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.writelines(new_lines)
            print(f"Modified: {file_path}")
            return True
        except Exception as e:
            print(f"Error writing {file_path}: {e}")
            return False
    
    return False

def find_test_files(directory):
    """Find all test files (.cpp files in tests/ directories)."""
    test_files = []
    
    for root, dirs, files in os.walk(directory):
        if 'test' in root.lower() or any('test' in d.lower() for d in Path(root).parts):
            for file in files:
                if file.endswith('.cpp') or file.endswith('.hpp'):
                    test_files.append(os.path.join(root, file))
    
    return test_files

def main():
    if len(sys.argv) > 1:
        directory = sys.argv[1]
    else:
        directory = "."
    
    print(f"Scanning for .find() patterns in test files under: {directory}")
    
    test_files = find_test_files(directory)
    
    if not test_files:
        print("No test files found!")
        return 1
    
    print(f"Found {len(test_files)} test files to process:")
    for f in test_files:
        print(f"  {f}")
    print()
    
    modified_count = 0
    
    for file_path in test_files:
        if process_file(file_path):
            modified_count += 1
    
    print(f"\nProcessed {len(test_files)} files, modified {modified_count} files")
    
    if modified_count > 0:
        print("\nNext steps:")
        print("1. Review the FIXME comments and convert to exact equality")
        print("2. Fix any FAIL() statements by using proper exact equality")
        print("3. Remove the comments once fixed")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())