# **Comprehensive Test Plan for a Stateless Bash-Like Shell**

## **1\. Introduction**

Test plan for a stateless bash-like shell. State limited to Current Working Directory (CWD) and environment variables. Excludes background jobs, shell variables, aliases, history. Focus: core operations, filesystem/environment interaction, error handling, CWD/env var integrity.

## **2\. Core Command Execution**

Tests command discovery, invocation, argument passing, and exit status. Relies on PATH and direct paths.

### **2.1. Command Invocation**

Test launching executables.

* **Via PATH:**
  * Standard utilities (e.g., ls, echo).
  * Variations: No args (ls), multiple args (ls \-l /tmp), substring name.
* **Via Absolute Path:**
  * Full path execution (e.g., /bin/echo message).
  * Variations: Standard utility (/usr/bin/wc), custom executable, path with symlinks.
* **Via Relative Path:**
  * Relative to CWD (e.g., ./my\_script, build/my\_app).
  * Variations: In CWD (./test\_exec), in subdir (subdir/test\_exec), using ../ (../bin/test\_exec).

### **2.2. Argument Passing**

Verify accurate argument transmission to programs.

* Simple arguments.
* Arguments with spaces/special characters (quoting tested in Section 10).
* Many arguments (up to system limits like ARG\_MAX).
* Empty arguments ("", '').
* Variations: Mix of simple/quoted args, max args, empty string arg.

### **2.3. Command Not Found**

Handle non-existent or non-executable commands.

* Non-existent command name.
* Variations: Simple non-existent (nonexistentcommand), with args (nonexistentcommand arg1), PATH incorrect.
* Expected: Error to stderr, non-zero exit status (e.g., 127).

### **2.4. Permissions**

Behavior for files lacking execute permissions.

* Attempt to execute file without execute permission.
* Variations: User-owned no-exec, other-owned no-exec, readable but not executable script.
* Expected: Error to stderr, non-zero exit status (e.g., 126).

### **2.5. Script Execution**

Ability to execute scripts with shebang lines (e.g., \#\!/bin/sh).

* Execute simple script (e.g., my\_script.sh with echo "hello").
* Variations: Common interpreter (\#\!/bin/sh), custom interpreter, script with args.

### **2.6. Command Termination and Exit Status**

Capture and reflect command exit statuses.

* Commands exiting 0 (success).
* Commands exiting non-zero (failure).
* Variations: true (exit 0), false (exit 1), custom script exit code (e.g., exit 42).

**Table 1: Core\_Command\_Execution\_Test\_Cases (Illustrative Examples)**

| Test Case ID | Description/Objective | Pre-conditions (CWD, Env Vars like PATH) | Command Line | Expected Stdout | Expected Stderr | Expected Exit Status | Variations (Changes in command, args, PATH, permissions) |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| CCE-001 | Execute command from PATH | PATH=/bin:/usr/bin, CWD arbitrary | ls | (listing) | (empty) | 0 | ls \-a, cat /etc/hostname, PATH=/usr/bin:/bin (order change) |
| CCE-002 | Execute command with absolute path | CWD arbitrary | /bin/pwd | (current path) | (empty) | 0 | /bin/echo test, /nonexistentpath/cmd (failure) |
| CCE-003 | Command not found | PATH=/bin | unknowncmd | (empty) | unknowncmd: command not found | 127 | unknowncmd arg1, PATH="" unknowncmd |
| CCE-004 | Execute permission denied | File noexec.sh exists, is not executable | ./noexec.sh | (empty) | ./noexec.sh: Permission denied | 126 | Absolute path to non-executable, relative path |
| CCE-005 | Script execution with shebang | script.sh (e.g. \#\!/bin/sh\\necho hello) is executable in CWD | ./script.sh | hello\\n | (empty) | 0 | Script with args: ./script.sh arg1, script with different shebang |
| CCE-006 | Command exits with specific non-zero status | PATH includes directory with test\_exit\_5.sh (which does exit 5\) | test\_exit\_5.sh | (empty) | (empty) | 5 | test\_exit\_1.sh (exits 1), command that fails due to internal logic |

## **3\. Input/Output Redirection**

Tests redirection of stdin, stdout, stderr. Redirections are per-command.

### **3.1. Standard Output Redirection (\> and \>\>)**

* command \> file: Create/truncate file with stdout.
  * Variations: New file (echo "output" \> out.txt), truncate existing (echo "new" \> out.txt), empty output (cmd\_no\_output \> out.txt).
* command \>\> file: Create/append to file with stdout.
  * Variations: New file (echo "append1" \>\> append.txt), append existing (echo "append2" \>\> append.txt), empty output (cmd\_no\_output \>\> append.txt).
* Path variations: CWD, relative (subdir/file), absolute (/tmp/file).
* Permissions: Redirect to no-write-permission location. Expected: error, command no-run, non-zero exit.

### **3.2. Standard Input Redirection (\<)**

* command \< file: Command reads stdin from file.
  * Variations: Small file (wc \-l \< input.txt), large file (cat \< large\_input.txt), empty file (cat \< empty\_input.txt).
* Special files: Input from /dev/null.
* Permissions: Redirect from no-read-permission file. Expected: error, command no-run, non-zero exit.
* Non-existent file: Redirect from non-existent file. Expected: error, command no-run, non-zero exit.

### **3.3. Standard Error Redirection (2\> and 2\>\>)**

* command 2\> file: Redirect stderr to file.
  * Variations: ls /nonexistent 2\> err.txt, cmd\_writes\_both\_streams 2\> err.txt.
* command 2\>\> file: Append stderr to file.
  * Variations: ls /nonexistent\_again 2\>\> err.txt.
* Permissions/path variations similar to stdout.

### **3.4. Combined Output and Error Redirection**

* command \> file 2\>&1 or command &\> file: Redirect stdout and stderr to same file (overwrite).
  * Variations: cmd\_writes\_both \> all\_out.txt 2\>&1, cmd\_writes\_only\_stdout \> all\_out.txt 2\>&1.
* command \>\> file 2\>&1 or command &\>\> file: Append stdout and stderr.
  * Variations: cmd\_writes\_both \>\> all\_append.txt 2\>&1.
* Order of redirection: Test cmd \> file 2\>&1 vs cmd 2\>&1 \> file.

### **3.5. Here Documents (\<\<HEREDOC)**

* Basic: cat \<\<EOF...EOF.
  * Variations: cat \<\<END\\nline1\\nEND, delimiter with special chars (cat \<\<"MY-DELIM"\\ncontent\\nMY-DELIM), empty here doc.
* Quoting delimiter (\<\<'EOF' vs \<\<EOF): Quoted suppresses expansions; unquoted allows (env vars, command sub).
* Unterminated here docs: Expected: wait for input, timeout, or syntax error.

### **3.6. File Descriptor Manipulation (Advanced, if supported)**

* command \>\&N: Stdout to FD N.
* command \<\&N: Stdin from FD N.
* command N\> file: FD N to file.
* command N\>&-: Close FD N.

### **3.7. Interaction with Pipes**

* cmd1 \> out | cmd2: Typically syntax error. Test shell reaction.
* cmd1 \< in | cmd2: cmd1 reads in, output to cmd2.
* cmd1 | cmd2 \> out: cmd2 output to out.

**Table 2: IO\_Redirection\_Test\_Cases (Illustrative Examples)**

| Test Case ID | Description/Objective | Pre-conditions (CWD, Files, Permissions) | Command Line | Expected Output (Terminal) | Expected File Content(s) (file:content) | Expected Stderr | Expected Exit Status | Variations (different files, permissions, command outputs) |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| IOR-001 | Stdout redirection (overwrite) | existing.txt contains "old" | echo "new" \> existing.txt | (empty) | existing.txt:new\\n | (empty) | 0 | Redirect to new file, redirect empty output (true \> empty.txt) |
| IOR-002 | Stdout redirection (append) | append.txt contains "line1\\n" | echo "line2" \>\> append.txt | (empty) | append.txt:line1\\nline2\\n | (empty) | 0 | Append to new file, append empty output |
| IOR-003 | Stdin redirection | input.txt contains "data\\n" | cat \< input.txt | data\\n | (N/A) | (empty) | 0 | Empty input file, large input file |
| IOR-004 | Stderr redirection | (none) | ls /nonexistent\_file 2\> err.log | (empty) | err.log:(error message)\\n | (empty, captured in file) | (non-zero from ls) | Append stderr (2\>\>), redirect stderr of a command that also produces stdout |
| IOR-005 | Combined stdout/stderr redirection | (none) | sh \-c 'echo out; echo err \>&2' \> combined.log 2\>&1 | (empty) | combined.log:out\\nerr\\n (order may vary) | (empty, captured in file) | 0 | Using &\>, redirecting a command that only produces stdout or only stderr |
| IOR-006 | Here document | (none) | cat \<\<EOF\\nHello\\nEOF | Hello\\n | (N/A) | (empty) | 0 | Quoted delimiter \<\<'EOF', here doc with env var expansion (if supported: echo "$HOME" inside) |
| IOR-007 | Redirection failure (permission) | no\_write\_dir/ exists, not writable | echo "test" \> no\_write\_dir/file | (empty) | (file not created/changed) | (permission denied message) | (non-zero from shell) | Read from unreadable file, redirect to non-existent path component (/nonexistent\_dir/subdir/file) |

## **4\. Piping (|)**

Tests connecting stdout of one command to stdin of another.

### **4.1. Simple Two-Command Pipes**

* command1 | command2 (e.g., ls \-l | wc \-l).
* Verify data flow.
* Exit status: Conventionally, last command's status.
* Variations: echo "one two three" | wc \-w, sleep 1 | echo "done", true | cat.

### **4.2. Multi-Command Pipes**

* command1 | command2 |... | commandN (e.g., cat file.txt | grep "pattern" | sort).
* Correct data flow through all stages.
* Performance with many stages.
* Variations: 3-stage pipe (echo "a\\nc\\nb" | sort | uniq), longer pipes (5+ commands).

### **4.3. Piping with Commands Producing No Output/Expecting No Input**

* true | command2: command2 gets EOF immediately.
* command1 | wc \-c (where command1 has no output): wc \-c reports 0\.
* echo "data" | consumer\_that\_reads\_nothing: Clean pipe setup/teardown.
* Variations: echo \-n "" | wc \-c, cat /dev/null | wc \-l.

### **4.4. Piping with Error Conditions in Intermediate Commands**

* command1\_fails | command2: Behavior of command2, overall exit status.
* command1\_succeeds\_but\_writes\_to\_stderr | command2: command2 should not get command1's stderr.
* command1 | command2\_fails: Overall exit status reflects command2 failure.
* Variations: cat non\_existent\_file | wc \-l, echo "data" | grep \--invalid-option "pattern", sh \-c 'exit 1' | sh \-c 'exit 2'.

### **4.5. Piping Involving Built-ins**

Test built-ins (if any beyond cd/pwd) in pipelines.

* echo "foo" | some\_builtin or some\_builtin | wc \-l.
* pwd | cat: pwd stdout correctly piped. cd in subshell for pipe component doesn't affect parent shell CWD.

### **4.6. Large Data Transfers through Pipes**

* Pipe commands with significant data (e.g., dd... | wc \-c) to test buffering, flow control, completeness.

### **4.7. Interaction with Redirection**

* cmd1 \> file | cmd2: Often syntax error. Test shell reaction.
* cmd1 \< file | cmd2: cmd1 reads file, its stdout to cmd2.
* cmd1 | cmd2 \> file\_out: cmd2 output redirected.
* cmd1 | cmd2 2\> file\_err: cmd2 stderr redirected. cmd1 stderr to terminal.

**Table 3: Piping\_Test\_Cases (Illustrative Examples)**

| Test Case ID | Description/Objective | Pre-conditions (CWD, Env Vars, Input Files/Data) | Command Line (Pipeline) | Expected Stdout (of last command) | Expected Stderr (from any command, specify which) | Expected Exit Status (of pipeline) | Variations (different commands, data sizes, error conditions in pipe segments) |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| PIPE-001 | Simple two-command pipe | (none) | \`echo "hello world" \\ | wc \-w\` | 2\\n | (empty) | 0 |
| PIPE-002 | Multi-command pipe | data.txt contains "b\\na\\nc\\na" | \`cat data.txt \\ | sort \\ | uniq \-c\` | 2 a\\n 1 b\\n 1 c\\n | (empty) |
| PIPE-003 | First command fails | (none) | \`ls /nonexistent\_path \\ | wc \-l\` | 0\\n (if wc gets EOF) | ls:... No such file... (from ls) | 0 (from wc) |
| PIPE-004 | Last command fails | (none) | \`echo "data" \\ | grep \--invalid-option\` | (empty or error from grep) | grep: invalid option... (from grep) | (non-zero from grep) |
| PIPE-005 | Pipe with redirection at end | (none) | \`echo "line1\\nline2" \\ | wc \-l \> count.txt\` | (empty) | (empty) | 0 |
| PIPE-006 | Large data transfer | (none) | \`dd if=/dev/zero bs=1k count=1k \\ | wc \-c(adjustdd\` for system) | 1024000\\n (or similar) | (dd messages to stderr, if any) | 0 |

## **5\. Command Chaining (Sequential Execution)**

Tests sequential command execution, possibly conditional. CWD/env vars for each command depend on preceding commands in the same line.

### **5.1. Semicolon (;) \- Unconditional Sequential Execution**

* command1; command2: command2 runs regardless of command1's status.
* Multiple commands: cmd1; cmd2; cmd3.
* CWD changes: mkdir newdir\_sc; cd newdir\_sc; pwd (pwd reflects newdir\_sc).
* Temporary env vars: VAR=val cmd1; cmd2 (VAR=val only for cmd1).
* Variations: true; echo "Next", false; echo "Still Next", pwd; cd /tmp; pwd; cd \-; pwd.

### **5.2. Logical AND (&&) \- Conditional Execution on Success**

* command1 && command2: command2 runs if command1 exits 0\.
* Chain: cmd1 && cmd2 && cmd3.
* command1 fails: command2 not executed.
* Exit status: First failure's status, or last command's if all succeed.
* Variations: true && echo "Success", false && echo "No Success", mkdir testdir\_and && cd testdir\_and && pwd.

### **5.3. Logical OR (||) \- Conditional Execution on Failure**

* command1 | | command2: command2 runs if command1 exits non-zero.
* Chain: cmd1 | | cmd2 | | cmd3 (left-associative).
* command1 succeeds: command2 not executed.
* Exit status: First success's status, or last command's if all fail.
* Variations: false | | echo "Failure recovery", true | | echo "Not executed", cd /nonexistent\_dir | | echo "cd failed".

### **5.4. Combinations of &&, ||, and ;**

* Precedence/associativity (&&, || equal, left-associative, tighter than ;).
  * true && false | | echo "runs" (prints "runs").
  * false && echo "no" | | echo "yes" (prints "yes").
  * cmd1; cmd2 && cmd3 | | cmd4.
  * cmd1 | | cmd2 && cmd3.
* Variations: true; false && echo "A" | | echo "B" (Expected: B), mkdir test\_combo && cd test\_combo | | echo "Failed mkdir/cd"; echo "Done".

### **5.5. Impact of CWD and Environment Variables Across Chains**

* cd /tmp && pwd (pwd shows /tmp).
* cd /nonexistent\_dir | | echo "cd failed"; pwd (pwd shows original CWD).
* VAR=foo cmd1 && VAR=bar cmd2 (cmd1 sees VAR=foo, cmd2 sees VAR=bar).
* Built-in export ENV\_VAR=value: export MYVAR=initial; (false && export MYVAR=secondary); printenv MYVAR (MYVAR remains initial).

**Table 4: Command\_Chaining\_Test\_Cases (Illustrative Examples)**

| Test Case ID | Description/Objective | Pre-conditions (CWD, Env Vars) | Command Line (Chain) | Expected CWD after chain (if changed) | Expected Env Vars after chain (if changed by a built-in) | Expected Stdout/Stderr (overall) | Expected Exit Status (of entire chain) | Variations (different success/failure of commands, CWD changes, env var settings) |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| CHAIN-001 | Semicolon: unconditional | CWD=/usr | cd /tmp; pwd | /tmp | Unchanged | /tmp\\n | 0 | false; echo "OK" (stdout "OK\\n", exit 0 from echo) |
| CHAIN-002 | Logical AND: success path | CWD=/home/user | mkdir test\_and && cd test\_and | /home/user/test\_and | Unchanged | (empty) | 0 | true && true && echo "All good" |
| CHAIN-003 | Logical AND: failure path | CWD=/home/user | false && echo "Not printed" | /home/user | Unchanged | (empty) | 1 (from false) | cd /nonexistent && echo "Not printed" |
| CHAIN-004 | Logical OR: success path | CWD=/home/user | \`true \\ | \\ | echo "Not printed"\` | /home/user | Unchanged | (empty) |
| CHAIN-005 | Logical OR: failure path | CWD=/home/user | \`false \\ | \\ | echo "Printed"\` | /home/user | Unchanged | Printed\\n |
| CHAIN-006 | Mixed operators | CWD=/home/user | \`true && false \\ | \\ | echo "Fallback"; pwd\` | /home/user | Unchanged | Fallback\\n/home/user\\n |
| CHAIN-007 | Env var scoping in chain | CWD=/home/user, INITIAL\_VAR=present | TEMP\_VAR=val1 sh \-c 'printenv TEMP\_VAR'; sh \-c 'printenv TEMP\_VAR INITIAL\_VAR' | /home/user | INITIAL\_VAR=present | val1\\n\\npresent\\n (TEMP\_VAR not in second cmd) | 0 | TEMP\_VAR=v1 cmd1 && TEMP\_VAR=v2 cmd2 |

## **6\. Environment Variable Management**

Tests shell interaction with environment variables (one of two persistent states).

### **6.1. Displaying Environment Variables**

* env (external utility or built-in).
* printenv.
* printenv VAR\_NAME.
* printenv NONEXISTENT\_VARIABLE (empty stdout, newline, non-zero exit).
* Variations: printenv PATH, printenv NONEXISTENT\_VARIABLE, env.

### **6.2. Setting Environment Variables for a Single Command**

* VAR=value command: VAR set only for command.
* VAR=value VAR2=value2 command: Multiple vars for command.
* Command should observe these vars.
* Subsequent commands (VAR=val cmd1; cmd2) should not see VAR=val for cmd2.
* Empty values: VAR= command (VAR is empty string).
* Special chars in values: MYVAR="hello world" my\_script.
* Variations: TEST\_VAR=test\_value sh \-c 'echo $TEST\_VAR', EMPTY\_VAR= sh \-c 'printenv EMPTY\_VAR; echo end', SPACED\_VAR="two words" sh \-c 'echo $SPACED\_VAR'.

### **6.3. Impact of Standard Environment Variables**

* PATH: Command discovery. Test PATH="/tmp:$PATH" my\_command.
* HOME: cd behavior. Test HOME=/some/test/path cd; pwd.
* PWD: If shell maintains PWD env var, verify accuracy and inheritance.
* OLDPWD: If cd \- uses OLDPWD env var, test its setting and usage.
* Others (USER, TERM, LANG): Verify inheritance.
* Variations: HOME=/tmp cd; pwd, PATH=/dev/null ls.

### **6.4. Environment Inheritance**

* Child processes inherit copy of shell's environment.
* Child's env changes (e.g., export FOO=bar in script) must not affect parent shell.
* Variations: Shell has MY\_PARENT\_VAR=1. Execute sh \-c 'echo $MY\_PARENT\_VAR'. Execute sh \-c 'export CHILD\_EXPORT=child\_val'; printenv CHILD\_EXPORT (parent shell should not have CHILD\_EXPORT).

### **6.5. Handling of Special Characters in Variable Names and Values**

* Names: Valid POSIX chars (letters, digits, underscore, not starting with digit). Test invalid names for VAR=val cmd.
* Values: Spaces, quotes, newlines, \*, ?, $. MYVAR="foo\*bar" my\_script (script sees foo\*bar literally).
* Variations: SPEC\_VAL='val with spaces and \*' sh \-c 'echo "$SPEC\_VAL"', 1VAR=fail cmd (invalid name).

### **6.6. Maximum Number/Size of Environment Variables**

* Stress test: many vars or large vars for a command. Check system limits (ARG\_MAX), crashes.

## **7\. Current Directory Management (Built-ins cd, pwd)**

Tests cd and pwd for managing CWD (one of two persistent states).

### **7.1. pwd (Print Working Directory)**

* Basic: pwd.
* Output: Must reflect actual CWD.
* pwd after various cd ops.
* Options (pwd \-L, pwd \-P if supported). If not, consistent logical/physical reporting.
* pwd in /, deeply nested dirs.
* Variations: mkdir /tmp/test\_pwd; cd /tmp/test\_pwd; pwd, cd /; pwd.

### **7.2. cd \<path\> (Change Directory)**

* Absolute paths: cd /usr/bin.
* Relative paths: cd../local, cd./include.
* cd. (stay in CWD, update OLDPWD if used).
* cd.. (parent directory).
* Chained cd: mkdir /tmp/d1/d2; cd /tmp/d1/d2; cd..; pwd (should be /tmp/d1).
* Paths with symlinks: ln \-s /usr/bin /tmp/symlink\_dir; cd /tmp/symlink\_dir. CWD/PWD env var consistency.
* Variations: cd /var/log, mkdir \-p /tmp/a/b/c; cd /tmp/a/b/c; cd../..; pwd, cd. then pwd.

### **7.3. cd (to HOME)**

* cd (no args): To HOME dir.
* HOME not set/empty: Implementation-defined (error or no change).
* HOME invalid/non-existent: Error, non-zero exit, CWD unchanged.
* HOME is /.
* Variations: HOME=/tmp cd; pwd, unset HOME; cd, HOME=/nonexistent\_home\_dir cd.

### **7.4. cd \- (to previous directory)**

* Relies on OLDPWD env var (if supported). cd must update OLDPWD.
* Sequence: cd /tmp; cd /usr; cd \- (last cd \- to /tmp). Shell prints new CWD.
* OLDPWD not set: Error, CWD unchanged.
* Repeated cd \-: Toggle between dirs.
* Variations: mkdir /tmp/dir1 /tmp/dir2; cd /tmp/dir1; cd /tmp/dir2; cd \-; pwd, initial shell OLDPWD not set; cd \-, cd /nonexistent\_dir (fails), then cd \-.

### **7.5. Error Handling for cd**

* cd to non-existent dir: Error, non-zero exit, CWD unchanged.
* cd to a file: Error, non-zero exit, CWD unchanged.
* cd with permission issues: Error, non-zero exit, CWD unchanged.
* Variations: cd /this/path/does/not/exist, touch /tmp/a\_file; cd /tmp/a\_file, mkdir /tmp/no\_access\_dir; chmod 000 /tmp/no\_access\_dir; cd /tmp/no\_access\_dir.

### **7.6. cd and PWD/OLDPWD Environment Variables**

* If shell maintains PWD/OLDPWD env vars, they must be updated correctly by successful cd.
* Verify PWD/OLDPWD after cd., cd.., cd /path, cd (HOME), cd \-, symlink cd.
* If cd fails, PWD/OLDPWD must remain unchanged.

**Table 5: CWD\_Management\_Test\_Cases (Illustrative Examples)**

| Test Case ID | Description/Objective | Pre-conditions (Initial CWD, Env Vars like HOME, OLDPWD, PWD, Directory Structure, Permissions) | Command Line (cd/pwd) | Expected CWD After Command | Expected PWD Env Var After Command | Expected OLDPWD Env Var After Command | Expected Stdout/Stderr | Expected Exit Status | Variations (different paths, symlinks, permissions, HOME/OLDPWD states) |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| CWD-001 | pwd basic functionality | Initial CWD: /home/user | pwd | /home/user | /home/user (if PWD updated by shell startup) | (as set) | /home/user\\n | 0 | cd /tmp; pwd |
| CWD-002 | cd absolute path | Initial CWD: /home/user, PWD=/home/user, OLDPWD=/previous | cd /var/log | /var/log | /var/log | /home/user | (empty) | 0 | cd /, cd to deeply nested path |
| CWD-003 | cd relative path | Initial CWD: /usr, PWD=/usr, OLDPWD=/home | cd./bin (assuming /usr/bin exists) | /usr/bin | /usr/bin | /usr | (empty) | 0 | cd../lib, cd. |
| CWD-004 | cd to HOME | Initial CWD: /tmp, HOME=/home/testuser (exists) | cd | /home/testuser | /home/testuser | /tmp | (empty) | 0 | HOME not set, HOME is invalid path |
| CWD-005 | cd \- functionality | Initial CWD: /usr, PWD=/usr, OLDPWD=/tmp | cd \- | /tmp | /tmp | /usr | /tmp\\n | 0 | OLDPWD not set, cd \- twice |
| CWD-006 | cd failure (non-existent) | Initial CWD: /tmp, PWD=/tmp, OLDPWD=/home | cd /nonexistent\_path | /tmp | /tmp | /home | (error message)\\n | (non-zero) | cd to a file, cd to no-permission dir |
| CWD-007 | cd with symlink (logical path) | CWD: /tmp, ln \-s /usr/bin /tmp/bin\_link, PWD=/tmp, OLDPWD=/prev | cd /tmp/bin\_link | /tmp/bin\_link (logical) or /usr/bin (physical) \- define expected | (matches CWD) | /tmp | (empty) | 0 | cd /tmp/bin\_link/.. (behavior depends on logical/physical tracking) |

## **8\. Error Handling and Exit Statuses**

Tests shell's robustness in handling its own errors and reporting command exit statuses.

### **8.1. Exit Status for Successful Commands**

* Commands exiting 0 (e.g., true). Shell's last exit status is 0\.
* Variations: true, sh \-c 'exit 0'.

### **8.2. Exit Status for Failed Commands (Shell-Detected or Command-Reported)**

* **Command not found:** Shell returns specific non-zero (e.g., 127).
* **Command not executable (permission denied):** Shell returns specific non-zero (e.g., 126).
* **Command fails internally:** Shell reports command's actual exit status.
* Variations: nonexistent\_command, ./non\_executable\_script, sh \-c 'exit 33'.

### **8.3. Exit Status Propagation (Summary for Compound Commands)**

* **Pipes (cmd1 | cmd2):** Last command's status.
* **&& chains:** Status of first failing command, or last if all succeed.
* **|| chains:** Status of first succeeding command, or last if all fail.
* **; chains:** Status of last command executed.
* Variations: Combine with commands that succeed/fail.

### **8.4. Shell's Behavior on Syntax Errors**

Parser identifies and handles malformed command lines.

* Malformed redirection: echo \>, echo \< \> foo.
* Malformed piping: | command, command |.
* Unmatched quotes/parentheses (if supported).
* Invalid command name chars.
* Expected: Error to stderr, non-zero shell exit status, no part of command executed.
* Variations: echo hello \>, ls |, echo 'unterminated quote.

### **8.5. Errors During Redirection Setup**

If redirection fails, command should not run.

* command \> /non\_writable\_dir/file: Redirection fails (permissions). Command no-run. Shell error status.
* command \< /non\_existent\_file: Redirection fails (missing file). Command no-run. Shell error status.
* Variations: cat \< /dev/null/nonexistent, echo test \> /root/protected\_file.

### **8.6. Consistency of Error Messages**

* Clear, concise, to stderr.
* Consistent formatting/terminology for similar errors.
* Source of error (shell or command) should be clear.

## **9\. Wildcard/Globbing**

Tests filename expansion (globbing). Based on filenames in CWD or specified path.

### **9.1. \* (Matches any sequence of characters, including none)**

* echo \*: All non-dotfiles in CWD.
* echo a\*, echo \*b, echo a\*b, echo \*.\*.
* \* in paths: echo /etc/\*/\*.conf.
* Variations: Empty dir, one file, many files, diverse names.

### **9.2. ? (Matches any single character)**

* echo?.txt: Matches a.txt, not ab.txt.
* echo f??: Matches foo, not f.
* Variations: Matching, partially matching, non-matching names.

### **9.3. \`\` (Matches any one of the enclosed characters or range)**

* echo \[abc\].txt: Matches a.txt, b.txt, c.txt.
* echo \[a-c\].txt: Matches a.txt, b.txt, c.txt.
* echo \[\!abc\].txt or echo \[^abc\].txt: Matches d.txt, not a.txt.
* echo \[a-zA-Z\]\*: Files starting with a letter.
* Character classes (if supported, e.g., \[\[:alpha:\]\]): echo \[\[:digit:\]\]\*.log.
* Variations: Various char sets, ranges, negations.

### **9.4. Combinations of Glob Patterns**

* echo a\*.\[ch?\].
* echo /usr/bin/\[a-c\]\*z\*.
* Variations: Combine \*, ?, \`\`.

### **9.5. Globbing in Different Argument Positions**

* echo \* (glob is only arg).
* cat \*.txt (glob provides multiple args).
* my\_command arg1 \*.c arg2 (glob is intermediate arg).
* Variations: Commands accepting variable numbers of args.

### **9.6. No Match Behavior**

* If glob matches no files (e.g., echo non\_existent\_pattern\*):
  * POSIX default: Pattern passed as literal argument. Test this shell's behavior.
* Variations: \*, ?, \`\` patterns with no matches.

### **9.7. Dotfiles (. files)**

* \*, ?, \[a-z\] should not match dotfiles unless dot is explicit (e.g., echo.\*) or pattern is . or ...
* Test echo.\* (lists dotfiles).
* Test echo \* (no dotfiles).
* Variations: Dir with dotfiles and non-dotfiles.

### **9.8. Escaping Glob Characters**

* Literal \*, ?, \[:
  * echo \\\*file (output \*file).
  * echo 'file?' (output file?).
  * echo "file\[" (output file\[).
* Test with quotes (single, double) and backslash.

### **9.9. Limits (Number of files, argument length)**

* Globbing in dir with very many matching files.
* Shell should handle "Argument list too long" error gracefully, not crash.
* Variations: Gradually increase matching files.

## **10\. Quoting Mechanisms**

Tests single quotes ('...'), double quotes ("..."), backslashes (\\) for controlling interpretation of special characters and expansions (env var, command sub).

### **10.1. Single Quotes ('...')**

* All chars literal. No expansions.
* echo 'Hello World' \-\> Hello World.
* echo 'My $PATH is $PATH' \-\> My $PATH is $PATH.
* echo 'This is a \* test' \-\> This is a \* test.
* Embedded single quote: echo 'An apostrophe '"'"'s here'. Test shell's handling.
* Empty: echo '' (empty string arg).
* Variations: echo 'Value is $(date)', echo ';|&\>\<\*?'.

### **10.2. Double Quotes ("...")**

* Allows some expansions, most other chars literal.
  * Env var expansion: MYVAR=test echo "Var is $MYVAR" \-\> Var is test. Unset $NONEXISTENT\_VAR \-\> empty string.
  * Command substitution: echo "Date is $(date)" (if $(...) supported).
  * Backslash (\\): Special for \\", \\\\, \\$, \\\`.
* echo "Hello World" \-\> Hello World.
* echo "This is a \* test" \-\> This is a \* test (asterisk literal).
* echo "An escaped quote \\" here" \-\> An escaped quote " here.
* Empty: echo "" (empty string arg).
* Variations: KNOWN\_ENV=present echo "Value: $KNOWN\_ENV, Unset: $ABSENT\_ENV", echo "Output of ls: $(ls)", echo "Literal dollar: \\\\$HOME".

### **10.3. Backslash (\\)**

* Escapes next char, making it literal if special.
* echo Hello\\ World \-\> Hello World.
* echo \\\* \-\> \*.
* echo \\$HOME \-\> $HOME.
* echo \\\\ \-\> \\.
* Backslash at EOL (line continuation): echo hello \\ world. Test if supported.
* Variations: echo \\\>\\&\\|\\;, echo foo\\\\bar, backslash before non-special char (\\a).

### **10.4. Combinations and Nesting**

* echo 'Single quotes with "double" inside' \-\> Single quotes with "double" inside.
* echo "Double quotes with 'single' inside" \-\> Double quotes with 'single' inside.
* KNOWN\_ENV=val echo "Escaped \\\\$KNOWN\_ENV, but expanded $KNOWN\_ENV".
* Variations: Complex arg structures with helper script.

### **10.5. Behavior with Unmatched Quotes**

* echo 'this is open
* echo "this is also open
* Expected: Detect, secondary prompt (if multi-line supported) or syntax error. No hang/partial execution.
* Variations: Unmatched quote at EOI, unmatched quote then other tokens.

### **10.6. Quoting and Special Characters in Arguments**

* my\_command "arg with spaces" 'another arg with \*'.
* Ensure commands receive args exactly as specified. Use test script to print args.

## **11\. Command Substitution (e.g., $(command) or \`command\`)**

Tests executing a sub-command and substituting its stdout into the main command line. Sub-command inherits CWD/env, doesn't alter parent shell state.

### **11.1. Basic Command Substitution**

* $(command): echo Today is $(date).
* \`command\` (if supported): echo Today is \`date\`.
* Substitution into arg: touch file\_$(date \+%Y%m%d).txt.
* Variations: echo "Path is $(pwd)", echowhoami\`\`, ls $(echo /tmp).

### **11.2. Nested Command Substitution**

* echo Outer $(echo Inner $(date)) (if supported).
* Test nesting depth limits.
* Variations: echo $(echo $(echo nested)), echo $(cat $(echo filename.txt)).

### **11.3. Substitution with Commands Producing Multi-Line Output**

* echo "$(ls \-1)": Double-quoted, newlines preserved in single arg.
* echo $(ls \-1): Unquoted, output subject to word splitting and globbing.
* Variations: lines.txt with "a\\nb\\nc". Compare echo "$(cat lines.txt)" vs echo $(cat lines.txt). my\_arg\_printer $(cat lines.txt) vs my\_arg\_printer "$(cat lines.txt)".

### **11.4. Substitution with Commands Producing Special Characters or Whitespace**

* my\_command "$(echo 'hello world \*')": hello world \* as single arg, \* literal.
* my\_command $(echo 'hello world \*'): Word splitting, \* globbed.
* Variations: Output with tabs, leading/trailing spaces.

### **11.5. Substitution Resulting in Empty Output**

* echo "Output: $(true)" \-\> Output: \\n.
* echo "Output: $(grep non\_existent\_pattern file)".
* Argument parsing: cmd1 $(empty\_output) arg2 \-\> cmd1 arg2. cmd1 "$(empty\_output)" arg2 \-\> cmd1 "" arg2.
* Variations: my\_arg\_printer A $(true) B, my\_arg\_printer A "$(true)" B.

### **11.6. Trailing Newlines from Command Output**

* POSIX: Command substitution removes all trailing newlines. $(echo foo; echo bar) \-\> foo\\nbar.
* Test: echo \-n "$(echo 'line\\n')" \-\> line.
* Variations: Output with one, multiple, or no trailing newlines.

### **11.7. Error Conditions in Substituted Command**

* echo $(non\_existent\_command): Substitutes empty string? Outer command runs? Error from sub-command to stderr?
* echo $(cat /permission\_denied\_file): Similar.
* Exit status of sub-command: Does not directly affect main command line's status.
* Variations: Sub-command exits non-zero with/without output.

### **11.8. Interaction with Quoting**

* echo '$(date)' \-\> literal $(date).
* echo "\\$(date)" \-\> literal $(date).
* echo "$(echo "nested text with spaces")".
* Variations: Complex quote combinations around/within command substitutions.

## **12\. Signal Handling (Basic)**

Tests shell's response to signals, mainly SIGINT (Ctrl+C). No background jobs.

### **12.1. Ctrl+C (SIGINT) to Interrupt a Foreground Command**

* Run long-running command (e.g., sleep 100). Press Ctrl+C.
* Expected: Command terminates. Shell regains control, new prompt. Shell itself not terminated.
* Variations: Interrupt sleep 60, cat (waiting stdin), pipeline yes | head.

### **12.2. Shell's Behavior After Command Interruption**

* Prompt displayed correctly.
* Shell ready for new commands.
* CWD/env vars unaffected.
* Exit status: For interrupted command, 128 \+ signal\_number (e.g., 130 for SIGINT).
* Variations: Check CWD/env var after interrupt.

### **12.3. Rapid Ctrl+C Presses**

* Multiple Ctrl+C quickly.
* Expected: Graceful handling, no crash/inconsistent state/shell termination. Command terminates, one prompt.

### **12.4. Ctrl+C When No Command is Running (At Empty Prompt)**

* At empty prompt, Ctrl+C.
* Expected: Ignore or redisplay prompt. Shell not terminated.

### **12.5. Ctrl+D (EOF) at Empty Prompt**

* At empty prompt, Ctrl+D.
* Expected: Shell interprets as EOF, terminates gracefully (exit 0).

### **12.6. Ctrl+D When Input is Expected**

* During here doc input (cat \<\<EOF): Ctrl+D on new line terminates here doc.
* After opening quote (if multi-line input supported): echo 'abc. Ctrl+D might terminate input (syntax error) or be ignored.
* Variations: Ctrl+D in different input contexts.

### **12.7. Other Signals (Conditional on Scope)**

* Ctrl+\\ (SIGQUIT): If handled, terminate foreground command (maybe core dump), return to prompt. Shell not terminated. Exit status 128 \+ 3 \= 131\.
* Ctrl+Z (SIGTSTP): No background jobs. Shell ignores or passes to foreground process group. Shell should not suspend/background.

## **13\. Path Resolution and Command Discovery**

Focuses on PATH environment variable usage for locating commands.

### **13.1. PATH Traversal Order**

* PATH="/dir1:/dir2": If cmd in both, /dir1/cmd executed.
* Test with 3+ dirs in PATH, conflicting names.
* Variations: PATH=/tmp/p1:/tmp/p2 mycmd vs PATH=/tmp/p2:/tmp/p1 mycmd.

### **13.2. Empty PATH or PATH Not Set**

* PATH="" or unset: Only absolute/relative paths executable.
* ls (by name) should fail ("command not found").
* Variations: PATH="" ls, unset PATH; ls, PATH="" /bin/echo hello.

### **13.3. PATH with Invalid Entries**

* PATH="/dir1:/non\_existent\_dir:/dir2": Commands in /dir1, /dir2 found. Invalid entry skipped.
* PATH="/dir1:/not\_a\_dir\_file:/dir2": /not\_a\_dir\_file skipped.
* Variations: Invalid entry at start, middle, end of PATH.

### **13.4. PATH with Relative Directories**

* PATH=".:/usr/bin" or PATH="../bin:/usr/bin": Discovery uses paths relative to CWD.
* cd can change which command executes if . or relative paths in PATH.
* Variations: cd /tmp/rel\_test; PATH=".:/bin" my\_local, cd /tmp; PATH="rel\_test:/bin" my\_local.

### **13.5. PATH with Empty Components**

* PATH="/dir1::/dir2", PATH=":/dir1", PATH="/dir1:".
* POSIX: Empty component means CWD. Verify this.
* Variations: PATH=":/bin" my\_cwd\_cmd, PATH="/bin:" my\_cwd\_cmd, PATH="/bin::/usr/bin" my\_cwd\_cmd.

### **13.6. Caching (If Any)**

* If PATH lookup caching (unlikely for simple stateless shell):
  * Add new command to dir in PATH after shell start. Should be found.
  * Modify PATH (e.g., re-invoke shell). Change effective immediately.
* Simplest assumption: No caching.

### **13.7. Precedence of Built-ins over External Commands**

* If built-ins (e.g., built-in echo) same name as external in PATH, built-in runs.
* Test forcing external (e.g., /path/to/echo).
* Variations: Create /tmp/cd, put /tmp in PATH. cd still runs built-in.

### **13.8. Command Name with Slashes**

* If command name has / (e.g., ./mycmd, /bin/ls), PATH search NOT used. Execute directly.
* Variations: ./existing\_executable, /usr/bin/true, sub/dir/cmd, ./nonexistent\_cmd\_with\_slash (fail "No such file or directory").

## **14\. Conclusion**

This test plan covers a stateless bash-like shell (CWD, env vars are only state). Focus on integrity of CWD/env vars, per-command context for ops like redirection, and predictable behavior. Successful completion of these tests ensures confidence in shell correctness and robustness. Systematic execution with attention to edge cases and POSIX standards (where applicable) is key.
