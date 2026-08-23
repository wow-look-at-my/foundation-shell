package parser

import (
	"fmt"
	"regexp"
	"strings"

	"foundation-shell/internal/lexer"
)

// varRefPattern finds the $NAME and ${NAME} references in a word as
// written, using the same name rule as the expander.
var varRefPattern = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)

// referencedVars returns the variable names a pre-expansion word expands.
// Dropping each marked $ first removes the escaped ones: `\$X` is literal
// text, not a reference.
func referencedVars(raw string) []string {
	cleaned := strings.ReplaceAll(raw, string(lexer.EscapeMarker)+"$", "")
	matches := varRefPattern.FindAllStringSubmatch(cleaned, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

// assignedNames returns the variables one command sets. Two forms set a
// variable in the parent shell: a standalone assignment, whose sole word is
// NAME=VALUE, and export with NAME=VALUE arguments. Both are invisible to a
// later expansion in the same input, so both count here.
func assignedNames(words []classifiedToken) []string {
	if len(words) == 0 {
		return nil
	}
	if len(words) == 1 {
		if !words[0].wasSingleQuoted && assignmentPattern.MatchString(words[0].value) {
			name, _, _ := strings.Cut(words[0].value, "=")
			return []string{name}
		}
		return nil
	}
	if words[0].value != "export" {
		return nil
	}
	var names []string
	for _, w := range words[1:] {
		if assignmentPattern.MatchString(w.value) {
			name, _, _ := strings.Cut(w.value, "=")
			names = append(names, name)
		}
	}
	return names
}

// checkAssignmentThenUse rejects an input that assigns a variable and then
// expands it further along, which reads the value from before the input ran
// (see ErrAssignmentThenUse). It walks commands in order, testing each
// command's references against the variables EARLIER commands assigned, so
// a word never counts as referencing its own assignment.
//
// One input assigns a handful of variables at most, so the seen list is a
// slice and the lookup is a scan.
func checkAssignmentThenUse(tokens []classifiedToken) error {
	var assigned []string

	for i := 0; i < len(tokens); {
		// words are the command's own words, which decide what it assigns.
		// refs additionally covers redirection targets: `> $OUT` expands the
		// stale value too, and it is a filename, so a wrong one is worse
		// than a wrong argument.
		var words, refs []classifiedToken
		for i < len(tokens) && !isChainOperator(tokens[i].tokenType) {
			if isRedirectionOperator(tokens[i].tokenType) {
				if i+1 < len(tokens) && !tokens[i+1].tokenType.IsOperator() {
					refs = append(refs, tokens[i+1])
					i += 2
					continue
				}
				// A dangling redirection; buildChain reports it.
				i++
				continue
			}
			words = append(words, tokens[i])
			refs = append(refs, tokens[i])
			i++
		}
		i++

		for _, w := range refs {
			if w.wasSingleQuoted {
				// Single quotes suppress expansion, so the name is data —
				// `sh -c 'echo $X'` hands $X to the child untouched.
				continue
			}
			for _, name := range referencedVars(w.rawValue) {
				for _, seen := range assigned {
					if seen == name {
						return fmt.Errorf("%w: %s", ErrAssignmentThenUse, name)
					}
				}
			}
		}

		assigned = append(assigned, assignedNames(words)...)
	}

	return nil
}
