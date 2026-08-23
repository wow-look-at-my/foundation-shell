# Expansion pipeline conformance.
#
# Spec: foundation-shell-spec src/expansion.md.

sandbox:
	network: false

tests:
	# expansion.md Word Splitting: an unquoted token that expands to empty
	# stays in the argument list as an EMPTY argument (POSIX drops it).
	- desc: an unset variable stays as an empty argument
	  cmd: "build/fsh-exec 'printf [%s] a $FSH_UNSET_VAR b'"
	  outputs:
		stdout:
			0: "^\\Q[a][][b]\\E$"

	# expansion.md Word Splitting: NO post-expansion word splitting -- a
	# value containing whitespace remains ONE argument.
	- desc: no post-expansion word splitting
	  cmd: "build/fsh-exec 'printf [%s] $X'"
	  inputs:
		env:
			X: "a b"
	  outputs:
		stdout:
			0: "^\\Q[a b]\\E$"

	# expansion.md Special Parameters: $? starts at 0; the whole input is
	# one parse, so a fresh fsh-exec sees the pre-input status.
	- desc: $? expands to 0 in a fresh shell
	  cmd: "build/fsh-exec 'echo $?'"
	  outputs:
		stdout:
			0: "^0$"

	# expansion.md Variable Expansion rule 4: the braced body is a VERBATIM
	# environment lookup, not the special parameter -- ${?} is empty.
	- desc: ${?} is a verbatim environment lookup, so it is empty
	  cmd: "build/fsh-exec 'printf [%s] ${?}'"
	  outputs:
		stdout:
			0: "^\\Q[]\\E$"

	# expansion.md NOT Supported: other POSIX special parameters stay
	# literal.
	- desc: $$ is literal
	  cmd: "build/fsh-exec 'echo $$'"
	  outputs:
		stdout:
			0: "^\\Q$$\\E$"

	- desc: $1 is literal
	  cmd: "build/fsh-exec 'echo $1'"
	  outputs:
		stdout:
			0: "^\\Q$1\\E$"

	# expansion.md Escaped Dollar Signs: \$ suppresses expansion via the
	# escape marker; the literal $ survives.
	- desc: an escaped dollar is literal
	  cmd: "build/fsh-exec 'echo \\$HOME'"
	  outputs:
		stdout:
			0: "^\\Q$HOME\\E$"

	# expansion.md Expansion Rules rule 1: name scanning is byte-wise ASCII
	# -- a non-ASCII character ends the name instead of extending it.
	- desc: variable names are ASCII-only
	  cmd: "build/fsh-exec 'echo $FSH_A\u00e9'"
	  inputs:
		env:
			FSH_A: x
	  outputs:
		stdout:
			0: "^x\u00e9$"

	# expansion.md Expansion Rules rule 2: greedy matching takes the
	# longest valid name.
	- desc: greedy name matching
	  cmd: "build/fsh-exec 'echo $ABC'"
	  inputs:
		env:
			A: a
			AB: ab
			ABC: abc
	  outputs:
		stdout:
			0: "^abc$"

	# expansion.md Examples: braces limit the name explicitly.
	- desc: braces limit the variable name
	  cmd: "build/fsh-exec 'echo ${A}BC'"
	  inputs:
		env:
			A: a
	  outputs:
		stdout:
			0: "^aBC$"

	# expansion.md Expansion Rules rules 5-6: ${} stays literal.
	- desc: empty braces stay literal
	  cmd: "build/fsh-exec 'echo ${}'"
	  outputs:
		stdout:
			0: "^\\Q${}\\E$"

	- desc: an unclosed brace stays literal
	  cmd: "build/fsh-exec 'echo ${FSH_UNCLOSED'"
	  outputs:
		stdout:
			0: "^\\Q${FSH_UNCLOSED\\E$"

	# expansion.md Spliced Values Are Protected: a variable value
	# containing substitution syntax is DATA -- never executed, never
	# re-expanded.
	- desc: a value holding $(...) is data, not code
	  cmd: "build/fsh-exec 'echo $X'"
	  inputs:
		env:
			X: "$(echo pwned)"
	  outputs:
		stdout:
			0: "^\\Q$(echo pwned)\\E$"

	- desc: a value holding backticks is data, not code
	  cmd: "build/fsh-exec 'echo $Y'"
	  inputs:
		env:
			Y: "`date`"
	  outputs:
		stdout:
			0: "^\\Q`date`\\E$"

	- desc: a value holding $HOME is data, not re-expanded
	  cmd: "build/fsh-exec 'echo $Z'"
	  inputs:
		env:
			Z: "$HOME"
	  outputs:
		stdout:
			0: "^\\Q$HOME\\E$"

	# expansion.md Non-Existent Variables: missing variables vanish inside
	# larger words.
	- desc: a missing braced variable expands to nothing inside a word
	  cmd: "build/fsh-exec 'echo prefix${MISSING_FSH_VAR}end'"
	  outputs:
		stdout:
			0: "^prefixend$"

	# expansion.md Tilde Expansion: ~user is NOT supported.
	- desc: tilde-user does not expand
	  cmd: "build/fsh-exec 'echo ~user'"
	  outputs:
		stdout:
			0: "^~user$"

	# expansion.md Tilde Expansion: a tilde not at the start of the token
	# does not expand.
	- desc: a tilde inside a token does not expand
	  cmd: "build/fsh-exec 'echo /path/~'"
	  outputs:
		stdout:
			0: "^\\Q/path/~\\E$"

	# expansion.md Tilde Expansion (whole-token granularity): quoting ANY
	# part of the token suppresses tilde expansion for the whole token.
	- desc: tilde suppression is whole-token
	  cmd: "build/fsh-exec 'echo ~/\"docs\"'"
	  inputs:
		env:
			HOME: /home/testuser
	  outputs:
		stdout:
			0: "^\\Q~/docs\\E$"

	# expansion.md Quoting and Expansion: double quotes allow variable
	# expansion; single quotes suppress it.
	- desc: double quotes allow variable expansion
	  cmd: "build/fsh-exec 'echo \"$FSH_V\"'"
	  inputs:
		env:
			FSH_V: hello
	  outputs:
		stdout:
			0: "^hello$"

	- desc: single quotes suppress variable expansion
	  cmd: "build/fsh-exec \"echo '\\$FSH_V'\""
	  inputs:
		env:
			FSH_V: hello
	  outputs:
		stdout:
			0: "^\\Q$FSH_V\\E$"
