# Diagnostic output conformance: the three-line caret block format on the
# real binaries.
#
# Spec: foundation-shell-spec src/diagnostics.md (2 format, 5 canonical
# strings, 6 multi-line input, 7 multiple errors, 8 examples).
#
# Where the spec fixes the number of lines, the block is captured to a file
# and matched with \A...\z, which pins the line count as well as the text.

sandbox:
	network: false

tests:
	# diagnostics.md 2, 8.3: each error is EXACTLY three lines -- the input
	# line, the caret line, and "error: <message>". The unclosed token
	# spans positions 5..11, so six carets start at column 5.
	- desc: an unclosed single quote produces exactly the three-line caret block
	  cmd: "build/fsh-exec \"echo 'hello\" 2> {outputs.err.txt}"
	  exit: 1
	  outputs:
		!stdout:
			- "hello"
		files:
			err.txt:
				match:
					- "\\A\\Qecho 'hello\\E\n\\Q     ^^^^^^\\E\n\\Qerror: unclosed single quote\\E\n\\z"

	# diagnostics.md 8.4: the double-quote variant.
	- desc: unclosed double quote caret block
	  cmd: "build/fsh-exec 'echo \"hello'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qecho \"hello\\E$"
			1: "^\\Q     ^^^^^^\\E$"
			2: "^\\Qerror: unclosed double quote\\E$"

	# diagnostics.md 7: multiple errors produce blank-line-separated blocks
	# -- the spec's normative two-error example. An unclosed substitution
	# containing an unclosed quote reports the quote first (innermost),
	# then the substitution.
	- desc: a two-error input produces two blank-line-separated blocks
	  cmd: "build/fsh-exec -c 'echo $(foo \"bar' 2> {outputs.err.txt}"
	  exit: 1
	  outputs:
		!stdout:
			- "echo"
		files:
			err.txt:
				match:
					- "\\A\\Qecho $(foo \"bar\\E\n\\Q     ^^^^^^^^^^\\E\n\\Qerror: unclosed double quote\\E\n\n\\Qecho $(foo \"bar\\E\n\\Q     ^^^^^^^^^^\\E\n\\Qerror: unclosed command substitution $(...)\\E\n\\z"

	# diagnostics.md 8.1: a trailing pipe gets one caret under the
	# operator, and the message carries no operator suffix.
	- desc: trailing pipe caret block
	  cmd: "build/fsh-exec 'echo hello |'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qecho hello |\\E$"
			1: "^\\Q           ^\\E$"
			2: "^\\Qerror: unexpected operator at end\\E$"

	# diagnostics.md 8.1: a two-character trailing operator gets two carets.
	- desc: a trailing && caret block spans the operator
	  cmd: "build/fsh-exec 'echo hello &&'"
	  exit: 1
	  outputs:
		stderr:
			1: "^\\Q           ^^\\E$"
			2: "^\\Qerror: unexpected operator at end\\E$"

	# diagnostics.md 8.2: a missing redirection target points at the
	# operator.
	- desc: missing redirection target caret block
	  cmd: "build/fsh-exec 'echo >'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qecho >\\E$"
			1: "^\\Q     ^\\E$"
			2: "^\\Qerror: missing redirection target\\E$"

	# diagnostics.md 5.1: leading-operator detection is REQUIRED of the
	# analyzer, with the operator in the message.
	- desc: leading operator caret block
	  cmd: "build/fsh-exec '| foo'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Q| foo\\E$"
			1: "^\\Q^\\E$"
			2: "^\\Qerror: unexpected operator at start: |\\E$"

	# diagnostics.md 6.4: for multi-line input the block shows the LINE
	# containing the error, with carets relative to that line's start.
	- desc: multi-line input reports the offending line
	  cmd: build/fsh
	  exit: 1
	  inputs:
		stdin: |
			echo hello
			cat "unclosed
	  outputs:
		stderr:
			0: "^\\Qcat \"unclosed\\E$"
			1: "^\\Q    ^^^^^^^^^\\E$"
			2: "^\\Qerror: unclosed double quote\\E$"

	# diagnostics.md 5.1 note / quoting.md 5.5: an EVEN quote count can be
	# unclosed -- the canonical strings carry no count-based suffix.
	- desc: an even quote count still reports unclosed, with no suffix
	  cmd: "build/fsh-exec \"echo 'a 'b\""
	  exit: 1
	  outputs:
		stderr:
			2: "^\\Qerror: unclosed single quote\\E$"
