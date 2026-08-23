# I/O redirection conformance.
#
# Spec: foundation-shell-spec src/redirection.md. Each test cites its
# section.
#
# The sandbox mounts the working directory read-only, so every target that
# the shell must actually create is a {outputs.X} path. Two tests need a
# writable cwd for a relative target and cd into a private temp directory.

sandbox:
	network: false

tests:
	# redirection.md 3.2: > truncates, so only the second write survives.
	- desc: the > operator truncates the target file
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo first > {outputs.out.txt}' && \"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo second > {outputs.out.txt}'"
	  outputs:
		files:
			out.txt:
				match:
					- "\\A\\Qsecond\\E\n\\z"

	# redirection.md 3.3: >> appends.
	- desc: the >> operator appends to the target file
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo line1 > {outputs.log.txt}' && \"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo line2 >> {outputs.log.txt}'"
	  outputs:
		files:
			log.txt:
				match:
					- "\\A\\Qline1\\E\n\\Qline2\\E\n\\z"

	# redirection.md 3.1: < replaces stdin with the file.
	- desc: the < operator redirects input
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'cat < {inputs.in.txt}'"
	  inputs:
		files:
			in.txt: |
				from-file
	  outputs:
		stdout:
			0: "^from-file$"

	# redirection.md 3.4: 2> captures stderr, leaving stdout alone.
	- desc: the 2> operator captures stderr
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'ls /nonexistent_fsh_dir 2> {outputs.err.txt}'; test $? -ne 0"
	  outputs:
		!stdout:
			0: "."
		files:
			err.txt:
				match:
					- "."

	# redirection.md 3.5: 2>> appends stderr, so two runs leave two lines.
	- desc: the 2>> operator appends stderr
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'ls /nonexistent_fsh_dir 2> {outputs.err.txt}'; \"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'ls /nonexistent_fsh_dir 2>> {outputs.err.txt}'; wc -l < {outputs.err.txt}"
	  outputs:
		stdout:
			0: "^ *2$"

	# redirection.md 5.3: last-wins at PARSE time -- the earlier target is
	# never opened, so f1.txt is NOT created (a documented POSIX deviation;
	# bash would create and truncate it).
	- desc: the last redirection wins and the earlier target is never created
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo test > {outputs.f1.txt} > {outputs.f2.txt}'"
	  outputs:
		files:
			f2.txt:
				match:
					- "\\A\\Qtest\\E\n\\z"
		!files:
			f1.txt:
				exists: true

	# redirection.md 9.4: a target that expands to empty is a loud parse
	# error; nothing on the line executes.
	- desc: an empty redirection target is a parse error
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo hi > $UNSET_FSH_VAR'"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		stderr:
			- "empty redirection target"

	# redirection.md 9.5: fd duplication is guarded with its own error.
	- desc: 2>&1 is rejected with the fd-duplication error
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo hi 2>&1'"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		stderr:
			- "file descriptor duplication is not supported: &1"

	- desc: the >&2 form is the same guard
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo hi >&2'"
	  exit: 1
	  outputs:
		stderr:
			- "file descriptor duplication is not supported: &2"

	# redirection.md 9.5: a QUOTED target is a legitimate filename.
	- desc: a quoted &1 target writes a file literally named &1
	  cmd: "F=\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\"; cd \"$(mktemp -d)\" && \"$F\" \"echo hi > '&1'\" && cat '&1'"
	  outputs:
		stdout:
			0: "^hi$"

	# redirection.md 9.5: the guard inspects the PRE-expansion token, so a
	# target that only becomes &1 through expansion is a legal filename.
	- desc: an expansion-produced &1 target is a legal filename
	  cmd: "F=\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\"; cd \"$(mktemp -d)\" && \"$F\" 'echo hi > $X' && cat '&1'"
	  inputs:
		env:
			X: "&1"
	  outputs:
		stdout:
			0: "^hi$"

	# redirection.md 9.1: runtime open failures print EXACTLY this message
	# -- no wrapper prefix, filename exactly once -- and fail only the
	# command.
	- desc: the input open failure message is canonical
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'cat < nonexistent.txt'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qcannot open input file nonexistent.txt: no such file or directory\\E$"

	# redirection.md 9.1: the chain CONTINUES after an open failure.
	- desc: the chain continues after an input open failure
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'cat < nonexistent.txt || echo recovered'"
	  outputs:
		stdout:
			0: "^recovered$"
		stderr:
			- "cannot open input file nonexistent.txt: no such file or directory"

	# redirection.md 8.3 + 9.1: a missing parent directory is an output
	# open failure with the canonical message.
	- desc: the output open failure message is canonical
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo x > missingdir/file.txt'"
	  exit: 1
	  outputs:
		stderr:
			0: "^\\Qcannot open output file missingdir/file.txt: no such file or directory\\E$"

	# redirection.md 9.3: a redirection with no target is a parse error.
	- desc: a missing redirection target is a parse error
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo >'"
	  exit: 1
	  outputs:
		stderr:
			- "missing redirection target"

	# redirection.md 9.3: an operator instead of a filename.
	- desc: a redirection followed by an operator is a parse error
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo > | cat'"
	  exit: 1
	  outputs:
		stderr:
			- "missing redirection target: > followed by operator |"

	# lexer.md 3.4: a redirection cannot be continued across a newline --
	# the separator is NOT suppressed after redirection operators, so the
	# parser rejects the dangling redirection instead of silently
	# redirecting into the next line's first word.
	- desc: a newline after a redirection is a parse error, not a continuation
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" -c $'echo hi >\\nout.txt'"
	  exit: 1
	  outputs:
		!stdout:
			0: "."
		stderr:
			- "missing redirection target: > followed by operator ;"

	# redirection.md 5.1: redirections may appear anywhere in the command.
	- desc: redirection position is independent
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" '< {inputs.in.txt} cat'"
	  inputs:
		files:
			in.txt: |
				data
	  outputs:
		stdout:
			0: "^data$"

	# redirection.md 12.3: builtins respect redirections.
	- desc: the pwd builtin respects output redirection
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'pwd > {outputs.where.txt}' && [ \"$(cat {outputs.where.txt})\" = \"$PWD\" ]"
