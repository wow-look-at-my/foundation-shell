# Command substitution conformance: $(...) and backticks.
#
# Spec: foundation-shell-spec src/expansion.md (Command Substitution).

sandbox:
	network: false

tests:
	- desc: simple command substitution
	  cmd: "build/fsh-exec 'echo $(echo hello)'"
	  outputs:
		stdout:
			0: "^hello$"

	- desc: command substitution resolves a path with which
	  cmd: "build/fsh-exec 'echo $(which echo)'"
	  outputs:
		stdout:
			0: "/echo$"

	- desc: command substitution with spaces in the body
	  cmd: "build/fsh-exec 'echo $(echo hello world)'"
	  outputs:
		stdout:
			0: "^hello world$"

	- desc: nested command substitution
	  cmd: "build/fsh-exec 'echo $(echo $(echo nested))'"
	  outputs:
		stdout:
			0: "^nested$"

	- desc: backtick substitution
	  cmd: "build/fsh-exec 'echo `echo backtick`'"
	  outputs:
		stdout:
			0: "^backtick$"

	- desc: backtick substitution with spaces
	  cmd: "build/fsh-exec 'echo `echo hello world`'"
	  outputs:
		stdout:
			0: "^hello world$"

	- desc: command substitution containing a pipe
	  cmd: "build/fsh-exec 'echo $(echo hello | tr a-z A-Z)'"
	  outputs:
		stdout:
			0: "^HELLO$"

	- desc: multiple command substitutions in one word list
	  cmd: "build/fsh-exec 'echo $(echo one) $(echo two)'"
	  outputs:
		stdout:
			0: "^one two$"

	- desc: command substitution in the middle of a word
	  cmd: "build/fsh-exec 'echo prefix$(echo middle)suffix'"
	  outputs:
		stdout:
			0: "^prefixmiddlesuffix$"

	- desc: command substitution preserves the exit code on success
	  cmd: "build/fsh-exec 'echo $(true)'"

	# expansion.md Quoting and Expansion: single quotes suppress
	# substitution, double quotes do not.
	- desc: single quotes prevent command substitution
	  cmd: "build/fsh-exec \"echo '\\$(echo hello)'\""
	  outputs:
		stdout:
			0: "^\\Q$(echo hello)\\E$"

	- desc: double quotes allow command substitution
	  cmd: "build/fsh-exec 'echo \"$(echo hello)\"'"
	  outputs:
		stdout:
			0: "^hello$"

	# wc pads its count, so the assertion allows the leading whitespace.
	- desc: command substitution capturing wc output
	  cmd: "build/fsh-exec 'echo $(echo -n test | wc -c)'"
	  outputs:
		stdout:
			0: "^ *4$"

	- desc: pwd inside a command substitution
	  cmd: "build/fsh-exec 'echo $(pwd)'"
	  outputs:
		stdout:
			0: "^/"
