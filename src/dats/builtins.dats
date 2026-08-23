# Builtin command and exit-code conformance.
#
# Spec: foundation-shell-spec src/execution.md (Builtin Commands, Standard
# Exit Codes, exit sentinel semantics).

sandbox:
	network: false

tests:
	# execution.md cd, Builtin Commands: builtins run in-process, so cd
	# persists for later commands in the same input.
	- desc: cd changes the directory for the rest of the chain
	  cmd: "build/fsh-exec 'cd / ; pwd'"
	  outputs:
		stdout:
			0: "^/$"

	# execution.md cd: failure message and status 1.
	- desc: cd failure message and status
	  cmd: "build/fsh-exec 'cd /nonexistent_fsh_dir'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qcd: /nonexistent_fsh_dir: no such file or directory\\E$"

	# execution.md pwd: prints the working directory, status 0.
	- desc: pwd prints the current directory
	  cmd: "[ \"$(build/fsh-exec pwd)\" = \"$PWD\" ]"

	# execution.md exit: a numeric argument becomes the shell's status.
	- desc: exit 7 exits with status 7
	  cmd: "build/fsh-exec 'exit 7'"
	  exit: 7

	# execution.md exit: the value is taken modulo 256, non-negative.
	- desc: exit 300 wraps to 44
	  cmd: "build/fsh-exec 'exit 300'"
	  exit: 44

	- desc: exit -1 wraps to 255
	  cmd: "build/fsh-exec 'exit -1'"
	  exit: 255

	# execution.md exit: at top level the chain STOPS.
	- desc: exit stops the sequence
	  cmd: "build/fsh-exec 'exit 7 ; echo after'"
	  exit: 7
	  outputs:
		!stdout:
			- "after"

	# execution.md exit: inside a pipeline segment, exit only sets that
	# command's status; the shell survives and the chain continues.
	- desc: exit inside a pipeline does not stop the shell
	  cmd: "build/fsh-exec 'exit 7 | cat ; echo after'"
	  outputs:
		stdout:
			0: "^after$"

	# execution.md exit: a non-numeric argument is a usage error (2) and
	# the shell does NOT exit.
	- desc: exit with a non-numeric argument is a usage error
	  cmd: "build/fsh-exec 'exit abc'"
	  exit: 2
	  outputs:
		stderr:
			0: "^\\Qexit: abc: numeric argument required\\E$"

	- desc: the shell survives a non-numeric exit argument
	  cmd: "build/fsh-exec 'exit abc ; echo survived'"
	  outputs:
		stdout:
			0: "^survived$"

	# execution.md Standard Exit Codes: command not found is 127.
	- desc: command not found is 127 with the canonical message
	  cmd: "build/fsh-exec 'nosuchcmd_fsh_test'"
	  exit: 127
	  outputs:
		stderr:
			0: "^\\Qnosuchcmd_fsh_test: command not found\\E$"

	# execution.md Standard Exit Codes: found but not executable is 126.
	# The fixture is written without an execute bit.
	- desc: a non-executable file is 126
	  cmd: build/fsh-exec {inputs.notexec.sh}
	  exit: 126
	  inputs:
		files:
			notexec.sh: |
				#!/bin/sh
				echo no
	  outputs:
		stderr:
			0: "\\Q/notexec.sh: permission denied\\E$"

	# execution.md Standard Exit Codes: a child killed by signal N reports
	# 128+N (SIGKILL = 9 -> 137), never -1 or 255.
	- desc: signal death reports 128+N
	  cmd: "build/fsh-exec \"sh -c 'kill -9 \\$\\$'\""
	  exit: 137

	# execution.md export: sets the variable; children see it.
	- desc: export sets a variable that children see
	  cmd: "build/fsh-exec \"export FSH_TEST_EXPORT=bar ; sh -c 'echo \\$FSH_TEST_EXPORT'\""
	  outputs:
		stdout:
			0: "^bar$"

	# execution.md export: no arguments prints the environment as
	# NAME=value lines.
	- desc: export with no arguments prints the environment
	  cmd: "build/fsh-exec 'export FSH_TEST_PRINTED=hello ; export'"
	  outputs:
		stdout:
			- "FSH_TEST_PRINTED=hello"

	# execution.md export: invalid names error but processing CONTINUES
	# with the remaining arguments.
	- desc: export continues past invalid names
	  cmd: "build/fsh-exec \"export FSH_A=1 1BAD=2 FSH_B=3 ; sh -c 'echo \\$FSH_A\\$FSH_B'\""
	  outputs:
		stdout:
			0: "^13$"
		stderr:
			- "export: invalid name: 1BAD=2"

	# execution.md export: the final status of a failed export is 1.
	- desc: export of an invalid name alone exits 1
	  cmd: "build/fsh-exec 'export 1BAD=2'"
	  exit: 1

	# execution.md Standalone Assignment: NAME=VALUE alone sets the
	# variable; children started later in the chain see it.
	- desc: a standalone assignment is visible to children
	  cmd: "build/fsh-exec \"FSH_TEST_ASSIGN=baz ; sh -c 'echo \\$FSH_TEST_ASSIGN'\""
	  outputs:
		stdout:
			0: "^baz$"

	# execution.md Standalone Assignment: a single-quoted word is NOT an
	# assignment -- it runs a command literally named FOO=bar.
	- desc: a single-quoted FOO=bar is a command, not an assignment
	  cmd: "build/fsh-exec \"'FSH_Q=bar'\""
	  exit: 127
	  outputs:
		stderr:
			0: "^\\QFSH_Q=bar: command not found\\E$"

	# execution.md Standalone Assignment: prefix assignments are NOT
	# supported -- two words make it a command named VAR=x.
	- desc: a prefix assignment VAR=x cmd is not supported
	  cmd: "build/fsh-exec 'FSH_P=x true'"
	  exit: 127
	  outputs:
		stderr:
			0: "^\\QFSH_P=x: command not found\\E$"

	# execution.md help: must list every builtin (export included) and the
	# $? parameter, and must NOT advertise a background & operator.
	- desc: help lists export and $? and no background operator
	  cmd: build/fsh-exec help
	  outputs:
		stdout:
			- "export"
			- "$?"
		!stdout:
			- "background"

	# execution.md clear: writes the ANSI clear-screen sequence.
	- desc: clear writes the ANSI clear sequence
	  cmd: build/fsh-exec clear
	  outputs:
		stdout:
			0: "^\\x1b\\[2J\\x1b\\[H$"
