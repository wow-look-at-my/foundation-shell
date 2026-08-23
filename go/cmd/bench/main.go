// Command bench runs shell benchmarks using hyperfine and outputs markdown.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"text/template"
	"time"
)

//go:embed results.md.tmpl
var resultsTmpl string

type Shell struct {
	Name string
	Path string
	Args func(cmd string) []string
}

type Benchmark struct {
	Name        string
	Category    string
	Description string
	Command     string
}

type HyperfineResult struct {
	Results []struct {
		Command string  `json:"command"`
		Mean    float64 `json:"mean"`
		Stddev  float64 `json:"stddev"`
		Min     float64 `json:"min"`
		Max     float64 `json:"max"`
	} `json:"results"`
}

type BenchResult struct {
	Shell  string
	Mean   time.Duration
	Stddev time.Duration
	Min    time.Duration
	Max    time.Duration
}

type BenchmarkResult struct {
	Benchmark
	Results []BenchResult
}

type Category struct {
	Name       string
	Benchmarks []BenchmarkResult
}

type TemplateData struct {
	Timestamp  string
	Shells     []string
	Categories []Category
}

var defaultBenchmarks = []Benchmark{
	// Startup benchmarks
	{
		Name:        "Empty Command",
		Category:    "Startup",
		Description: "Time to start the shell and exit immediately.",
		Command:     "true",
	},
	{
		Name:        "Simple Echo",
		Category:    "Startup",
		Description: "Time to start and run `echo hello`.",
		Command:     "echo hello",
	},
	// Execution benchmarks
	{
		Name:        "Simple Pipeline",
		Category:    "Execution",
		Description: "Two-command pipeline: `echo hello | cat`",
		Command:     "echo hello | cat",
	},
	{
		Name:        "Long Pipeline",
		Category:    "Execution",
		Description: "Four-command pipeline: `echo hello | cat | cat | cat`",
		Command:     "echo hello | cat | cat | cat",
	},
	{
		Name:        "Logical AND",
		Category:    "Execution",
		Description: "Command with logical AND: `true && echo success`",
		Command:     "true && echo success",
	},
	{
		Name:        "Logical OR",
		Category:    "Execution",
		Description: "Command with logical OR: `false || echo fallback`",
		Command:     "false || echo fallback",
	},
}

func main() {
	fshPath := flag.String("fsh", "", "path to fsh-exec binary (required)")
	warmup := flag.Int("warmup", 3, "warmup runs")
	runs := flag.Int("runs", 10, "benchmark runs")
	flag.Parse()

	if *fshPath == "" {
		fmt.Fprintln(os.Stderr, "error: -fsh flag is required")
		os.Exit(1)
	}

	// Define shells
	shells := []Shell{
		{Name: "fsh", Path: *fshPath, Args: func(cmd string) []string { return []string{"-c", cmd} }},
	}

	// Add available shells
	if path, err := exec.LookPath("bash"); err == nil {
		shells = append(shells, Shell{Name: "bash", Path: path, Args: func(cmd string) []string { return []string{"-c", cmd} }})
	}
	if path, err := exec.LookPath("zsh"); err == nil {
		shells = append(shells, Shell{Name: "zsh", Path: path, Args: func(cmd string) []string { return []string{"-c", cmd} }})
	}
	if path, err := exec.LookPath("pwsh"); err == nil {
		shells = append(shells, Shell{Name: "pwsh", Path: path, Args: func(cmd string) []string { return []string{"-NoProfile", "-Command", cmd} }})
	}

	// Run benchmarks
	categoryOrder := []string{"Startup", "Execution"}
	categoryMap := make(map[string][]BenchmarkResult)
	for _, bench := range defaultBenchmarks {
		result := BenchmarkResult{Benchmark: bench}

		// Build hyperfine command
		args := []string{
			"--warmup", fmt.Sprintf("%d", *warmup),
			"--runs", fmt.Sprintf("%d", *runs),
			"--export-json", "/dev/stdout",
		}

		for _, shell := range shells {
			shellArgs := shell.Args(bench.Command)
			fullCmd := shell.Path
			for _, arg := range shellArgs {
				fullCmd += " " + shellQuote(arg)
			}
			args = append(args, "-n", shell.Name, fullCmd)
		}

		// Run hyperfine
		cmd := exec.Command("hyperfine", args...)
		cmd.Stderr = os.Stderr
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: benchmark %q failed: %v\n", bench.Name, err)
			continue
		}

		// Parse results
		var hfResult HyperfineResult
		if err := json.Unmarshal(output, &hfResult); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse results for %q: %v\n", bench.Name, err)
			continue
		}

		for _, r := range hfResult.Results {
			result.Results = append(result.Results, BenchResult{
				Shell:  shellNameFromCommand(r.Command, shells),
				Mean:   time.Duration(r.Mean * float64(time.Second)),
				Stddev: time.Duration(r.Stddev * float64(time.Second)),
				Min:    time.Duration(r.Min * float64(time.Second)),
				Max:    time.Duration(r.Max * float64(time.Second)),
			})
		}

		categoryMap[bench.Category] = append(categoryMap[bench.Category], result)
	}

	// Build ordered categories
	var categories []Category
	for _, name := range categoryOrder {
		if benchmarks, ok := categoryMap[name]; ok {
			categories = append(categories, Category{Name: name, Benchmarks: benchmarks})
		}
	}

	// Build template data
	shellNames := make([]string, len(shells))
	for i, s := range shells {
		shellNames[i] = s.Name
	}

	data := TemplateData{
		Timestamp:  time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		Shells:     shellNames,
		Categories: categories,
	}

	// Render template
	tmpl, err := template.New("results").Funcs(template.FuncMap{
		"formatDuration": formatDuration,
		"ratio": func(a, b time.Duration) float64 {
			if b == 0 {
				return 0
			}
			return float64(a) / float64(b)
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"getFshMean": func(results []BenchResult) time.Duration {
			for _, r := range results {
				if r.Shell == "fsh" {
					return r.Mean
				}
			}
			return 0
		},
	}).Parse(resultsTmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to parse template: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to execute template: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(buf.String())
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	// Simple quoting for shell commands
	return "'" + s + "'"
}

func shellNameFromCommand(cmd string, shells []Shell) string {
	for _, s := range shells {
		if len(cmd) >= len(s.Path) && cmd[:len(s.Path)] == s.Path {
			return s.Name
		}
	}
	return "unknown"
}

func formatDuration(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000
	if ms < 1 {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000)
	}
	return fmt.Sprintf("%.1fms", ms)
}
