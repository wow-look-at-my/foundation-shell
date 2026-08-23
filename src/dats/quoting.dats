# Quoting conformance.
#
# Spec: foundation-shell-spec src/quoting.md (depth-tracked nesting,
# suppression flags) and src/lexer.md (tokenization). Each test cites the
# section its expectation comes from.

sandbox:
	network: false

tests:
	# quoting.md 6.1: whitespace-delimited same-type quotes NEST; the
	# nested pair stays literally in the argument, and only the outermost
	# pair is stripped.
	- desc: nested single quotes stay literal
	  cmd: "build/fsh-exec \"echo 'outer 'inner' end'\""
	  outputs:
		stdout:
			0: "^\\Qouter 'inner' end\\E$"

	# quoting.md 6.1: nesting is not limited to one level.
	- desc: multi-level single-quote nesting
	  cmd: "build/fsh-exec \"echo 'l1 'l2 'l3' l2' l1'\""
	  outputs:
		stdout:
			0: "^\\Ql1 'l2 'l3' l2' l1\\E$"

	# quoting.md 6.2: the same rule for double quotes.
	- desc: nested double quotes stay literal
	  cmd: "build/fsh-exec 'echo \"outer \"inner\" end\"'"
	  outputs:
		stdout:
			0: "^\\Qouter \"inner\" end\\E$"

	# quoting.md 6.2: double-quote semantics inside a nested region are
	# unchanged -- $VAR still expands, and the nested quotes print.
	- desc: expansion still runs inside nested double quotes
	  cmd: "build/fsh-exec 'echo \"path \"$HOME\" here\"'"
	  inputs:
		env:
			HOME: /home/testuser
	  outputs:
		stdout:
			0: "^\\Qpath \"/home/testuser\" here\\E$"

	# quoting.md 6.1.1 (POSIX-identical anchors): attached closers close,
	# so 'a' 'b' is two arguments.
	- desc: two adjacent quoted words are two arguments
	  cmd: "build/fsh-exec \"printf '[%s]' 'a' 'b'\""
	  outputs:
		stdout:
			0: "^\\Q[a][b]\\E$"

	# quoting.md 6.1.1: 'a'b concatenates, and so does 'a''b'.
	- desc: a quoted word concatenates with the bare word after it
	  cmd: "build/fsh-exec \"echo 'a'b\""
	  outputs:
		stdout:
			0: "^ab$"

	- desc: two attached quoted words concatenate
	  cmd: "build/fsh-exec \"echo 'a''b'\""
	  outputs:
		stdout:
			0: "^ab$"

	# quoting.md 6.1.1 / 2.3: the close-escape-reopen dance for attached
	# apostrophes works exactly as in POSIX.
	- desc: the escaped-apostrophe dance produces one word
	  cmd: "build/fsh-exec \"echo 'don'\\''t'\""
	  outputs:
		stdout:
			0: "^\\Qdon't\\E$"

	# quoting.md 12.6 / lexer.md 12.6 (flagship interleave): every interior
	# quote closes, the segments concatenate, and WasSingleQuoted
	# suppresses expansion. The line lives in a fixture because its own
	# quoting leaves no readable way to spell it inline.
	- desc: the flagship interleave stays unexpanded
	  cmd: "build/fsh-exec -c \"$(cat {inputs.line.fsh})\""
	  inputs:
		files:
			line.fsh: "echo \"Hello, \"'\"'\"$USER\"'\"'\"!\""
	  outputs:
		stdout:
			0: "^\\QHello, \"$USER\"!\\E$"

	# quoting.md 13.3: when the nesting rule reads "nest" and depth never
	# returns to 0, the input fails LOUDLY with the canonical error.
	- desc: an unreturned single-quote nesting depth is a loud error
	  cmd: "build/fsh-exec \"echo 'hello 'world\""
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		stderr:
			- "error: unclosed single quote"

	# quoting.md 13.3: the POSIX close-then-concatenate idiom is an
	# unclosed double quote here (the second " NESTS).
	- desc: an unreturned double-quote nesting depth is a loud error
	  cmd: "build/fsh-exec 'echo \"Total: \"$N'"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		stderr:
			- "error: unclosed double quote"

	# lexer.md 4.5: empty quoted strings ARE arguments.
	- desc: an empty quoted word is an empty argument
	  cmd: "build/fsh-exec \"printf '[%s]' ''\""
	  outputs:
		stdout:
			0: "^\\Q[]\\E$"

	- desc: an empty quoted word keeps its place between other arguments
	  cmd: "build/fsh-exec 'printf \"[%s]\" a \"\" b'"
	  outputs:
		stdout:
			0: "^\\Q[a][][b]\\E$"

	# quoting.md 2.5: a single-quoted part ANYWHERE suppresses ALL
	# expansion for the whole token (whole-token granularity).
	- desc: whole-token suppression keeps $HOME literal
	  cmd: "build/fsh-exec \"echo 'a'\\$HOME\""
	  outputs:
		stdout:
			0: "^\\Qa$HOME\\E$"

	# lexer.md 3.2: the per-word flags reset at every boundary -- an empty
	# single-quoted word must not suppress the NEXT word's expansion.
	- desc: quote flags do not leak across words
	  cmd: "build/fsh-exec \"echo '' \\$HOME\""
	  inputs:
		env:
			HOME: /home/testuser
	  outputs:
		stdout:
			0: "^\\Q /home/testuser\\E$"

	# expansion.md Tilde Expansion: any quoting suppresses tilde expansion.
	- desc: a double-quoted tilde is literal
	  cmd: "build/fsh-exec 'echo \"~\"'"
	  outputs:
		stdout:
			0: "^~$"

	- desc: a single-quoted tilde is literal
	  cmd: "build/fsh-exec \"echo '~'\""
	  outputs:
		stdout:
			0: "^~$"

	# expansion.md Tilde Expansion: a bare unquoted ~ expands to $HOME.
	- desc: a bare tilde expands to HOME
	  cmd: "build/fsh-exec 'echo ~'"
	  inputs:
		env:
			HOME: /home/testuser
	  outputs:
		stdout:
			0: "^\\Q/home/testuser\\E$"

	- desc: a tilde-rooted path expands to HOME
	  cmd: "build/fsh-exec 'echo ~/docs'"
	  inputs:
		env:
			HOME: /home/testuser
	  outputs:
		stdout:
			0: "^\\Q/home/testuser/docs\\E$"

	# quoting.md 6.4 / 5.3: different-type quote characters inside an open
	# region are literal content.
	- desc: double quotes inside single quotes are literal
	  cmd: "build/fsh-exec \"echo 'say \\\"hello\\\"'\""
	  outputs:
		stdout:
			0: "^\\Qsay \"hello\"\\E$"

	- desc: a single quote inside double quotes is literal
	  cmd: "build/fsh-exec \"echo \\\"it's fine\\\"\""
	  outputs:
		stdout:
			0: "^\\Qit's fine\\E$"

	# quoting.md 3.4: escaped double quotes inside double quotes.
	- desc: escaped quotes inside double quotes
	  cmd: "build/fsh-exec 'echo \"say \\\"hi\\\"\"'"
	  outputs:
		stdout:
			0: "^\\Qsay \"hi\"\\E$"

	# quoting.md 2.1: no escape processing inside single quotes.
	- desc: a backslash is literal inside single quotes
	  cmd: "build/fsh-exec \"echo 'a\\nb'\""
	  outputs:
		stdout:
			0: "^\\Qa\\nb\\E$"
