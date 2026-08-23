# Pipeline execution conformance.
#
# Spec: foundation-shell-spec src/execution.md (Pipeline Execution, Early
# Exit Terminates Producers, Spawn Failures, Builtins in Pipelines) and
# src/operators.md (Pipe Operator).
#
# Potentially-hanging cases carry a timeout. dats kills the whole process
# group and fails the test, which the spec forbids the shell to provoke.

sandbox:
	network: false

tests:
	# execution.md Early Exit Terminates Producers (normative): yes | head
	# -1 prints y and exits immediately, status 0 -- it MUST NOT hang, and
	# the terminated producer is silent.
	- desc: yes piped into head -1 terminates with status 0
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'yes | head -1'"
	  timeout: 10
	  outputs:
		stdout:
			0: "^y$"
		# A line that does not exist cannot match, so this asserts stderr
		# is empty: the killed producer must not report anything.
		!stderr:
			0: "."

	# execution.md Early Exit: a finite producer bigger than the pipe
	# buffer.
	- desc: seq piped into head -2 terminates
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'seq 1 100000 | head -2'"
	  timeout: 10
	  outputs:
		stdout:
			0: "^1$"
			1: "^2$"

	# execution.md Early Exit: a consumer that never reads stdin must not
	# wedge the producer.
	- desc: a consumer that ignores stdin does not hang
	  cmd: "[ \"$(\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo x | pwd')\" = \"$PWD\" ]"
	  timeout: 10

	# execution.md Spawn Failures: the failing consumer reports 127 and the
	# pipeline completes.
	- desc: a failing consumer reports 127
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo hi | nosuchcmd_fsh_test'"
	  timeout: 10
	  exit: 127
	  outputs:
		stderr:
			- "nosuchcmd_fsh_test: command not found"

	# execution.md Spawn Failures: a failing producer is reported on stderr
	# but the exit status is still the rightmost command's (cat's 0).
	- desc: a failing producer reports the error but the pipeline exits 0
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'nosuchcmd_fsh_test | cat'"
	  timeout: 10
	  outputs:
		stderr:
			- "nosuchcmd_fsh_test: command not found"

	# execution.md Builtins in Pipelines: builtins execute IN-PROCESS even
	# as pipeline members -- cd mutates the parent shell (a deliberate
	# POSIX divergence; POSIX shells print the original directory).
	- desc: a builtin cd in a pipeline mutates the shell
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'cd / | cat ; pwd'"
	  timeout: 10
	  outputs:
		stdout:
			0: "^/$"

	# operators.md Pipe Operator: the pipeline's exit code is the RIGHTMOST
	# command's.
	- desc: the pipeline status comes from the rightmost command
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" \"true | false | sh -c 'exit 5'\""
	  exit: 5

	- desc: a failing producer does not change the pipeline status
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'false | true'"

	- desc: a failing consumer sets the pipeline status
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'true | false'"
	  exit: 1

	# operators.md Pipe Operator: data flows between concurrent commands.
	- desc: data flows through a pipeline
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo hello world | tr a-z A-Z'"
	  outputs:
		stdout:
			0: "^HELLO WORLD$"

	# operators.md Multi-Stage Pipelines.
	- desc: a multi-stage pipeline works end to end
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'printf \"b\\na\\nb\\n\" | sort | uniq'"
	  outputs:
		stdout:
			0: "^a$"
			1: "^b$"

	# redirection.md 7.1: explicit output redirection overrides the pipe --
	# the downstream command reads EOF, and the file gets the data. The
	# disconnected pipe must not hang (execution.md Early Exit).
	- desc: output redirection overrides the pipe
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'echo hi > {outputs.piped.txt} | cat'"
	  timeout: 10
	  outputs:
		!stdout:
			0: "."
		files:
			piped.txt:
				match:
					- "\\A\\Qhi\\E\n\\z"

	# redirection.md 7.4: a mid-pipeline input redirection disconnects that
	# command from the pipe; the upstream producer terminates instead of
	# blocking.
	- desc: input redirection inside a pipeline does not wedge the producer
	  cmd: "\"$GO_TOOLCHAIN_DATS_BUILD_DIR/fsh-exec\" 'yes | cat < {inputs.side.txt}'"
	  timeout: 10
	  inputs:
		files:
			side.txt: |
				file-data
	  outputs:
		stdout:
			0: "^file-data$"
