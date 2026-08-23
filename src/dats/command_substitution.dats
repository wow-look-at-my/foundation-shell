# Command substitution conformance: $(...) and backticks.
#
# Spec: foundation-shell-spec src/expansion.md (Command Substitution).

sandbox:
	network: false

tests:
	- desc: simple command substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo hello)'"
	  outputs:
		stdout:
			0: "^hello$"

	- desc: command substitution resolves a path with which
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(which echo)'"
	  outputs:
		stdout:
			0: "/echo$"

	- desc: command substitution with spaces in the body
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo hello world)'"
	  outputs:
		stdout:
			0: "^hello world$"

	- desc: nested command substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo $(echo nested))'"
	  outputs:
		stdout:
			0: "^nested$"

	- desc: backtick substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo `echo backtick`'"
	  outputs:
		stdout:
			0: "^backtick$"

	- desc: backtick substitution with spaces
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo `echo hello world`'"
	  outputs:
		stdout:
			0: "^hello world$"

	- desc: command substitution containing a pipe
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo hello | tr a-z A-Z)'"
	  outputs:
		stdout:
			0: "^HELLO$"

	- desc: multiple command substitutions in one word list
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo one) $(echo two)'"
	  outputs:
		stdout:
			0: "^one two$"

	- desc: command substitution in the middle of a word
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo prefix$(echo middle)suffix'"
	  outputs:
		stdout:
			0: "^prefixmiddlesuffix$"

	- desc: command substitution preserves the exit code on success
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(true)'"

	# expansion.md Quoting and Expansion: single quotes suppress
	# substitution, double quotes do not.
	- desc: single quotes prevent command substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" \"echo '\\$(echo hello)'\""
	  outputs:
		stdout:
			0: "^\\Q$(echo hello)\\E$"

	- desc: double quotes allow command substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo \"$(echo hello)\"'"
	  outputs:
		stdout:
			0: "^hello$"

	# wc pads its count, so the assertion allows the leading whitespace.
	- desc: command substitution capturing wc output
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(echo -n test | wc -c)'"
	  outputs:
		stdout:
			0: "^ *4$"

	- desc: pwd inside a command substitution
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo $(pwd)'"
	  outputs:
		stdout:
			0: "^/"
