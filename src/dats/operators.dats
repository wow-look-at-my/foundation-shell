# Chain operator conformance.
#
# Spec: foundation-shell-spec src/operators.md (semantics, skip
# propagation, error cases) and src/lexer.md 3.3 (tokenization).

sandbox:
	network: false

tests:
	# operators.md Execution Algorithm: full skip propagation -- a skipped
	# segment preserves the status, so BOTH echos are skipped and the chain
	# yields false's 1, silently.
	- desc: skip propagation makes false && echo a && echo b silent
	  cmd: "build/fsh-exec 'false && echo a && echo b'"
	  exit: 1
	  outputs:
		!stdout:
			- "a"
			- "b"

	# operators.md Execution Algorithm: the mirror case for ||.
	- desc: skip propagation makes true || echo a || echo b silent
	  cmd: "build/fsh-exec 'true || echo a || echo b'"
	  outputs:
		!stdout:
			- "a"
			- "b"

	# operators.md Execution Algorithm: the preserved failure status
	# reaches the || after the skipped segment.
	- desc: false && echo a || echo b runs only b
	  cmd: "build/fsh-exec 'false && echo a || echo b'"
	  outputs:
		stdout:
			0: "^b$"

	# operators.md Combining Operators.
	- desc: false || echo one && echo two prints both
	  cmd: "build/fsh-exec 'false || echo one && echo two'"
	  outputs:
		stdout:
			0: "^one$"
			1: "^two$"

	- desc: true && false || echo fallback prints fallback
	  cmd: "build/fsh-exec 'true && false || echo fallback'"
	  outputs:
		stdout:
			0: "^fallback$"

	# operators.md Exit Code Definition / execution.md Runtime Failures
	# Never Abort the Chain: a spawn failure (127) feeds operator logic
	# like any other non-zero exit.
	- desc: a spawn failure recovers through ||
	  cmd: "build/fsh-exec 'nosuchcmd_fsh_test || echo fallback'"
	  outputs:
		stdout:
			0: "^fallback$"
		stderr:
			- "nosuchcmd_fsh_test: command not found"

	# execution.md Runtime Failures Never Abort the Chain.
	- desc: a spawn failure does not stop a semicolon chain
	  cmd: "build/fsh-exec 'nosuchcmd_fsh_test ; echo next'"
	  outputs:
		stdout:
			0: "^next$"

	# operators.md Semicolon Operator: the exit code is the LAST command's.
	- desc: a semicolon chain takes the last command's status
	  cmd: "build/fsh-exec 'true ; false'"
	  exit: 1

	- desc: a semicolon chain ignores the earlier failure
	  cmd: "build/fsh-exec 'false ; true'"

	# operators.md Error Cases: a trailing semicolon is VALID.
	- desc: a trailing semicolon is valid
	  cmd: "build/fsh-exec 'echo hi ;'"
	  outputs:
		stdout:
			0: "^hi$"

	# operators.md Error Cases: consecutive operators are a parse error;
	# nothing on the line executes.
	- desc: consecutive operators are a parse error
	  cmd: "build/fsh-exec 'echo a ; ; echo b'"
	  exit: 1
	  outputs:
		!stdout:
			- "a"
			- "b"
		stderr:
			- "consecutive operators: ; followed by ;"

	# operators.md Error Cases: an operator cannot start the line.
	- desc: a leading operator is a parse error
	  cmd: "build/fsh-exec '| cat'"
	  exit: 1
	  outputs:
		stderr:
			- "unexpected operator at start: |"

	# operators.md Error Cases: chain operators cannot end the line.
	- desc: a trailing pipe is a parse error
	  cmd: "build/fsh-exec 'echo hi |'"
	  exit: 1
	  outputs:
		!stdout:
			- "hi"
		stderr:
			- "unexpected operator at end"

	# lexer.md 3.3.4: quoted operator characters are literal data.
	- desc: a single-quoted pipe is data
	  cmd: "build/fsh-exec \"echo '|'\""
	  outputs:
		stdout:
			0: "^\\Q|\\E$"

	- desc: a double-quoted pipe is data
	  cmd: "build/fsh-exec 'echo \"|\"'"
	  outputs:
		stdout:
			0: "^\\Q|\\E$"

	# lexer.md 3.3.4: escaped operator characters are literal.
	- desc: an escaped pipe is data
	  cmd: "build/fsh-exec 'echo \\|'"
	  outputs:
		stdout:
			0: "^\\Q|\\E$"

	# lexer.md 3.3.4: a quoted '>' as a grep pattern must NOT become a
	# redirection -- the input file survives untouched, so it still has
	# both of its lines afterwards.
	- desc: a quoted redirection character does not truncate grep's input
	  cmd: "build/fsh-exec \"grep '>' {inputs.gfile}\" && wc -l < {inputs.gfile}"
	  inputs:
		files:
			gfile: |
				a>b
				plain
	  outputs:
		stdout:
			0: "^\\Qa>b\\E$"
			1: "^2$"

	# lexer.md 3.3: operators do not need whitespace.
	- desc: a pipe works without surrounding whitespace
	  cmd: "build/fsh-exec 'echo hello|tr a-z A-Z'"
	  outputs:
		stdout:
			0: "^HELLO$"

	- desc: a semicolon works without surrounding whitespace
	  cmd: "build/fsh-exec 'echo a;echo b'"
	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# lexer.md 3.3.2: a lone & is a literal word character, not an
	# operator, so it survives inside a word.
	- desc: an ampersand inside a word is a literal character
	  cmd: "build/fsh-exec 'echo a&b'"
	  outputs:
		stdout:
			0: "^\\Qa&b\\E$"

	# operators.md Precedence: pipes bind tighter than logical operators.
	- desc: a pipe binds tighter than &&
	  cmd: "build/fsh-exec 'echo test | grep test && echo found'"
	  outputs:
		stdout:
			0: "^test$"
			1: "^found$"
