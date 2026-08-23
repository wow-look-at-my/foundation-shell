# Whole-input (batch) mode conformance: piped stdin, script files, and
# fsh-exec -c.
#
# Spec: foundation-shell-spec src/execution.md (Non-Interactive Mode,
# Script Execution, Single Command Execution) and src/lexer.md (3.4
# newline separators, 8 comments).

sandbox:
	network: false

tests:
	# execution.md Non-Interactive Mode: ALL of stdin is one input;
	# unquoted newlines separate commands.
	- desc: multi-line stdin executes command by command
	  cmd: build/fsh
	  inputs:
		stdin: |
			echo a
			echo b
	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# lexer.md 8: comments -- the shebang line included -- are removed by
	# the lexer, so a script file needs no special first-line handling.
	- desc: script file with shebang and comments executes
	  cmd: build/fsh {inputs.script.fsh}
	  inputs:
		files:
			script.fsh: |
				#!/usr/bin/env fsh
				# a comment
				echo from-script # trailing note
	  outputs:
		stdout:
			0: "^from-script$"

	# execution.md Non-Interactive Mode consequence 2: a parse error
	# ANYWHERE rejects the whole input -- the first line must NOT have run,
	# so its redirection target is never created.
	- desc: a parse error anywhere rejects the whole input
	  cmd: build/fsh {inputs.bad.fsh}
	  exit: 1
	  inputs:
		files:
			bad.fsh: |
				echo ran > {outputs.marker.txt}
				echo "unclosed
	  outputs:
		stderr:
			- "error: unclosed double quote"
		!files:
			marker.txt:
				exists: true

	# lexer.md 3.4: after a chain operator the newline is a CONTINUATION.
	- desc: newline after && continues the line
	  cmd: build/fsh
	  inputs:
		stdin: |
			echo a &&
			echo b
	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# lexer.md 3.4: the skipping half of the same rule.
	- desc: false && newline echo skips the second command
	  cmd: build/fsh
	  exit: 1
	  inputs:
		stdin: |
			false &&
			echo skipped
	  outputs:
		!stdout:
			- "skipped"

	# expansion.md Special Parameters (whole-input consequence): the entire
	# input is ONE parse, so every $? expands BEFORE anything runs -- it
	# sees the pre-input status (0), NOT the false's 1. This is a
	# documented deviation from POSIX shells, where the same script prints 1.
	- desc: whole-input $? sees the pre-input status
	  cmd: build/fsh
	  inputs:
		stdin: |
			false
			echo [$?]
	  outputs:
		stdout:
			0: "^\\Q[0]\\E$"

	# execution.md Non-Interactive Mode consequence 1: the shell has
	# already consumed stdin to EOF, so $(cat) reads nothing -- a command
	# can never steal script text, and the following line still executes.
	- desc: commands inherit stdin at EOF, so $(cat) is empty
	  cmd: build/fsh
	  inputs:
		stdin: |
			echo got:$(cat)
			echo second
	  outputs:
		stdout:
			0: "^got:$"
			1: "^second$"

	# execution.md exit: exit stops the sequence; the remaining commands do
	# not run and the shell exits with the given status.
	- desc: exit stops a batch sequence
	  cmd: build/fsh
	  exit: 3
	  inputs:
		stdin: |
			echo one
			exit 3
			echo two
	  outputs:
		stdout:
			0: "^one$"
		!stdout:
			- "two"

	# lexer.md 3.4: blank lines and comment-only lines collapse into a
	# single separator.
	- desc: blank lines and comment lines between commands are harmless
	  cmd: build/fsh
	  inputs:
		stdin: |

			# leading comment
			echo a


			# middle
			echo b

	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# execution.md Non-Interactive Mode: quoted strings may span lines in
	# whole-input mode (the newline is token content).
	- desc: quoted newlines are content in batch mode
	  cmd: build/fsh
	  inputs:
		stdin: |
			echo "a
			b"
	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# execution.md Single Command Execution: fsh-exec -c takes the next
	# argument as THE command line.
	- desc: fsh-exec -c executes its argument
	  cmd: "build/fsh-exec -c 'echo hi'"
	  outputs:
		stdout:
			0: "^hi$"

	# execution.md Single Command Execution: the -c string is one input
	# with whole-input semantics -- embedded newlines separate commands.
	- desc: the fsh-exec -c string may contain newlines
	  cmd: "build/fsh-exec -c $'echo a\\necho b'"
	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# fsh-exec usage: -c without an argument is a usage error, status 2.
	- desc: fsh-exec -c without an argument is a usage error
	  cmd: build/fsh-exec -c
	  exit: 2

	# execution.md Script Execution: a missing script file reports the
	# canonical message with status 1.
	- desc: missing script file message
	  cmd: build/fsh no-such-script.fsh
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qcannot open script file no-such-script.fsh: no such file or directory\\E$"
