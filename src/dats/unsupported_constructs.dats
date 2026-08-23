# Constructs this shell deliberately does not implement. Each one must FAIL
# LOUDLY at parse time instead of being absorbed as ordinary words, which
# is what makes them safe for an automated caller to type.
#
# Spec: foundation-shell-spec src/parser.md (Background Execution Is
# Guarded, Assignment Then Use Is Guarded), src/lexer.md (3.3.2 lone &
# lexes as a word), src/redirection.md (9.5 fd duplication, 9.6
# here-documents).

sandbox:
	network: false

tests:
	# The regression this file exists for. Before the guard, & became an
	# argument, so this ran the sleep in the FOREGROUND and the shell
	# blocked for the full 30 seconds. The timeout would catch the hang.
	- desc: background & fails fast instead of hanging the shell
	  cmd: "build/fsh-exec -c \"bash -c 'sleep 30' &\""
	  timeout: 5
	  exit: 1
	  outputs:
		stderr:
			2: "^\\Qerror: background execution is not supported\\E$"

	# The other half of the trap: everything after the & was swallowed too,
	# so the second command silently never ran. The block is exactly three
	# lines, and the caret sits under the & alone.
	- desc: background & does not swallow the rest of the line
	  cmd: "build/fsh-exec -c 'echo a & echo b' 2> {outputs.err.txt}"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		files:
			err.txt:
				match:
					- "\\A\\Qecho a & echo b\\E\n\\Q       ^\\E\n\\Qerror: background execution is not supported\\E\n\\z"

	# The line lives in a fixture because dats rejects a heredoc in cmd at
	# parse time, even one quoted as an argument to another program.
	- desc: a here-document is named as unsupported, not as a missing target
	  cmd: "build/fsh-exec -c \"$(cat {inputs.line.fsh})\""
	  exit: 1
	  inputs:
		files:
			line.fsh: "cat <<EOF"
	  outputs:
		stderr:
			0: "^\\Qcat <<EOF\\E$"
			1: "^\\Q    ^\\E$"
			2: "^\\Qerror: here-documents are not supported\\E$"

	# A here-string is three <, forming two adjacent pairs. It must report
	# ONCE: a doubled diagnostic trains the reader to skim.
	- desc: a here-string reports exactly one error
	  cmd: "build/fsh-exec -c \"$(cat {inputs.line.fsh})\" 2> {outputs.err.txt}"
	  exit: 1
	  inputs:
		files:
			line.fsh: "cat <<<word"
	  outputs:
		files:
			err.txt:
				match:
					- "\\A\\Qcat <<<word\\E\n[^\n]*\n\\Qerror: here-documents are not supported\\E\n\\z"

	- desc: fd duplication is still rejected
	  cmd: "build/fsh-exec -c 'echo hi 2>&1'"
	  exit: 1
	  outputs:
		stderr:
			- "file descriptor duplication is not supported"

	# No false positives: & is an ordinary word character everywhere except
	# as a word of its own.
	- desc: an ampersand inside a word is still literal
	  cmd: "build/fsh-exec -c 'echo a&b'"
	  outputs:
		stdout:
			0: "^\\Qa&b\\E$"

	- desc: a quoted ampersand is still literal
	  cmd: "build/fsh-exec -c 'echo \"a & b\"'"
	  outputs:
		stdout:
			0: "^\\Qa & b\\E$"

	- desc: an escaped ampersand is still literal
	  cmd: "build/fsh-exec -c 'echo x\\&'"
	  outputs:
		stdout:
			0: "^\\Qx&\\E$"

	- desc: a url query string survives
	  cmd: "build/fsh-exec -c 'echo \"http://h/p?a=1&b=2\"'"
	  outputs:
		stdout:
			0: "^\\Qhttp://h/p?a=1&b=2\\E$"

	- desc: the && operator is unaffected
	  cmd: "build/fsh-exec -c 'true && echo ok'"
	  outputs:
		stdout:
			0: "^ok$"

	# The whole input expands in one pass before anything runs, so a
	# variable assigned in it still holds its pre-input value everywhere in
	# it. Left alone this printed an empty line and reported SUCCESS, which
	# is the one failure here that corrupts a result instead of stopping
	# the caller.
	- desc: capture-then-use is rejected instead of yielding empty
	  cmd: "build/fsh-exec -c 'OUT=$(echo captured); echo $OUT'"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		stderr:
			0: "^\\Qparse error: variable is assigned and used in the same input: OUT\\E$"

	- desc: export then use in the same input is rejected
	  cmd: "build/fsh-exec -c 'export X=hi; echo $X'"
	  exit: 1
	  outputs:
		stderr:
			- "variable is assigned and used in the same input: X"

	# The assignment itself is real: it mutates the shell's environment,
	# and a child reading that environment sees it. Only expansion in the
	# same input is stale, so these two must keep working.
	- desc: an assignment is still visible to a child that reads the environment
	  cmd: "build/fsh-exec -c 'FSH_ENVCHECK=hi; printenv FSH_ENVCHECK'"
	  outputs:
		stdout:
			0: "^hi$"

	- desc: a single-quoted reference is data and still resolves in the child
	  cmd: "build/fsh-exec -c \"FSH_CHILD=hi; sh -c 'echo \\$FSH_CHILD'\""
	  outputs:
		stdout:
			0: "^hi$"

	- desc: an assignment with no later expansion is fine
	  cmd: "build/fsh-exec -c 'FSH_UNUSED=hi; echo done'"
	  outputs:
		stdout:
			0: "^done$"

	- desc: expanding a variable nobody assigned here is fine
	  cmd: "build/fsh-exec -c 'FSH_A=1; echo [$FSH_B]'"
	  outputs:
		stdout:
			0: "^\\Q[]\\E$"

	- desc: a single < input redirection is unaffected
	  cmd: "build/fsh-exec -c 'cat < {inputs.in.txt}'"
	  inputs:
		files:
			in.txt: |
				payload
	  outputs:
		stdout:
			0: "^payload$"
