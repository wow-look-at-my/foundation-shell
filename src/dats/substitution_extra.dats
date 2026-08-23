# Command substitution conformance beyond the basics in
# command_substitution.dats: the single-pass security property, quote
# handling in bodies, nested backticks, and failure semantics.
#
# Spec: foundation-shell-spec src/expansion.md (Command Substitution) and
# src/execution.md (Command Substitution Executor, exit).

sandbox:
	network: false

tests:
	# expansion.md Single-Pass Expansion: substitution OUTPUT is never
	# re-scanned -- data read from a file is printed verbatim, NOT
	# executed. This is a security property.
	- desc: a $(...) payload read from a file stays literal
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(cat {inputs.payload.txt})'"
	  inputs:
		files:
			payload.txt: |
				$(echo pwned)
	  outputs:
		stdout:
			0: "^\\Q$(echo pwned)\\E$"

	# expansion.md Single-Pass Expansion: backtick payloads too.
	- desc: a backtick payload read from a file stays literal
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(cat {inputs.payload.txt})'"
	  inputs:
		files:
			payload.txt: |
				`date`
	  outputs:
		stdout:
			0: "^\\Q`date`\\E$"

	# lexer.md 7.2 / expansion.md Recursive Execution: quotes in the body
	# are honored by the recursive parse -- the two spaces survive.
	- desc: double quotes inside a body preserve spacing
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo \"a  b\")'"
	  outputs:
		stdout:
			0: "^\\Qa  b\\E$"

	# lexer.md 7.1 rule 4: a quoted ) is body content, not the closing
	# delimiter.
	- desc: a single-quoted close-paren does not close the substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" \"echo \\$(echo ')')\""
	  outputs:
		stdout:
			0: "^\\Q)\\E$"

	- desc: a double-quoted close-paren does not close the substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo \")\")'"
	  outputs:
		stdout:
			0: "^\\Q)\\E$"

	# quoting.md 6.3: nested backticks NEST by the quote rule and execute
	# recursively -- no escaping needed (a deliberate POSIX divergence).
	- desc: nested backticks execute recursively
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo `echo `echo hi``'"
	  outputs:
		stdout:
			0: "^hi$"

	# expansion.md Recursive Execution: single quotes INSIDE the body
	# suppress expansion within it.
	- desc: single quotes inside a body suppress nested substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" \"echo \\$(echo '\\$(pwd)')\""
	  outputs:
		stdout:
			0: "^\\Q$(pwd)\\E$"

	# expansion.md Failure Semantics: a runtime failure inside the body
	# prints to stderr, the substitution yields the captured stdout (empty
	# here), and the line CONTINUES -- status comes from the outer command.
	- desc: a failing substitution yields empty output and the line continues
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(nosuchcmd_fsh_test)'"
	  outputs:
		stdout:
			0: "^$"
		stderr:
			- "nosuchcmd_fsh_test: command not found"

	# expansion.md Failure Semantics: the substituted command's exit status
	# is DISCARDED -- it never aborts or fails the surrounding line.
	- desc: the substitution exit status is discarded
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" \"echo x\\$(sh -c 'exit 3')y\""
	  outputs:
		stdout:
			0: "^xy$"

	# execution.md exit: exit inside a substitution stops only the
	# substitution's own sequence; the shell survives and both echos run.
	- desc: exit inside a substitution does not stop the shell
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo before $(exit 7) ; echo after'"
	  outputs:
		stdout:
			0: "^\\Qbefore \\E$"
			1: "^after$"

	# expansion.md Output Handling: trailing newlines are trimmed, and
	# interior newlines are preserved.
	- desc: trailing newlines are trimmed and interior newlines are kept
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo x$(printf \"a\\nb\\n\\n\")y'"
	  outputs:
		stdout:
			0: "^xa$"
			1: "^by$"

	# expansion.md Quoting and Expansion: double quotes do not suppress
	# command substitution.
	- desc: substitution runs inside double quotes
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo \"count: $(echo 5)\"'"
	  outputs:
		stdout:
			0: "^\\Qcount: 5\\E$"

	# expansion.md Failure Semantics: a body that fails to PARSE fails the
	# whole line as a parse error -- nothing executes.
	- desc: a substitution body parse error rejects the whole line
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo |) ; echo after'"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
