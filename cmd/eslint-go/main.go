// Command eslint-go is a Go implementation of the ESLint CLI for the rules this
// port implements. Its output (stylish by default) is byte-identical to
// `eslint` 8.57.0's for the same files and configuration, which is what makes it
// a port rather than a reimplementation.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	eslint "github.com/jclyons52/eslint-go"
	"github.com/jclyons52/eslint-go/formatters"
	"github.com/jclyons52/eslint-go/rules"
)

const version = "0.1.0"

const usage = `eslint-go — a Go port of ESLint (eslint 8.57.0 behaviour)

Usage: eslint-go [options] file.js [file.js] [dir]

Basic configuration:
  -c, --config path::String    Use configuration from this file (.eslintrc.json)
  --no-eslintrc                Disable use of configuration from .eslintrc.*
  --env [String]               Enable a config environment (node, browser, es6, ...)
  --global [String]            Define a global variable
  --ignore-pattern [String]    Patterns of files to ignore
  --no-ignore                  Disable use of ignore files and patterns
  --ext [String]               Additional file extensions to lint (e.g. .mjs,.cjs)

Output:
  -f, --format String          Use a specific output format: stylish | json | compact | unix
  -o, --output-file path       Write the report to a file instead of stdout
  --color, --no-color          Force enabling/disabling of color
  --quiet                      Report errors only
  --max-warnings Int           Number of warnings to trigger nonzero exit code (-1 to allow all)

Fixing problems:
  --fix                        Automatically fix problems
  --fix-dry-run                Automatically fix problems without saving the changes

Other:
  --stdin                      Lint code provided on <STDIN>
  --stdin-filename String      Specify filename to process STDIN as
  --list-rules                 Print the rules this port implements
  -v, --version                Output the version number
  -h, --help                   Show help

This port implements a subset of ESLint's built-in rules; ` + "`eslint-go --list-rules`" + `
prints the implemented set, and configurations that reference other rules report
"Definition for rule 'x' was not found." exactly as ESLint does.
`

type options struct {
	patterns       []string
	configPath     string
	noEslintrc     bool
	format         string
	outputFile     string
	ext            string
	ignorePatterns []string
	globals        []string
	envs           []string
	noIgnore       bool
	fix            bool
	fixDryRun      bool
	quiet          bool
	color          bool
	noColor        bool
	maxWarnings    int
	stdin          bool
	stdinFilename  string
	listRules      bool
	versionOut     bool
	help           bool
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n\n%s", err, usage)
		return 2
	}
	if opts.help {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if opts.versionOut {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if opts.listRules {
		return listRules(stdout)
	}

	linter := eslint.NewLinter(rules.All()...)
	ruleMap := rules.Map()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	// Collect targets.
	var files []string
	if opts.stdin {
		files = nil
	} else if len(opts.patterns) == 0 {
		fmt.Fprintf(stderr, "Error: you must provide at least one file, directory or pattern to lint\n\n%s", usage)
		return 2
	} else {
		ignore := newGlobMatcher(opts.ignorePatterns)
		if !opts.noIgnore {
			patterns := append([]string{}, opts.ignorePatterns...)
			patterns = append(patterns, ignorePatternsFromConfig(filepath.Join(cwd, "linted.js"))...)
			patterns = append(patterns, defaultIgnorePatterns()...)
			if extra := readIgnoreFile(filepath.Join(cwd, ".eslintignore")); len(extra) > 0 {
				patterns = append(patterns, extra...)
			}
			ignore = newGlobMatcher(patterns)
		}
		exts := defaultExtensions
		if opts.ext != "" {
			for _, e := range strings.Split(opts.ext, ",") {
				e = strings.TrimSpace(e)
				if e == "" {
					continue
				}
				if !strings.HasPrefix(e, ".") {
					e = "." + e
				}
				exts = append(exts, e)
			}
		}
		files, err = collectFiles(opts.patterns, exts, ignore, cwd)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		if len(files) == 0 {
			fmt.Fprintf(stderr, "Error: No files matching the pattern %q were found.\n", strings.Join(opts.patterns, " "))
			return 2
		}
	}

	// Lint.
	var results []*eslint.LintResult
	if opts.stdin {
		source, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 2
		}
		name := opts.stdinFilename
		if name == "" {
			name = "<text>"
		}
		cfg, cerr := configFor(name, opts, ruleMap)
		if cerr != nil {
			fmt.Fprintf(stderr, "Error: %v\n", cerr)
			return 2
		}
		applyCLIConfig(cfg, opts)
		filename := name
		if name == "<text>" {
			filename = ""
		} else if abs, aerr := filepath.Abs(name); aerr == nil {
			// ESLint reports the resolved path for --stdin-filename.
			filename = abs
		}
		result := linter.Lint(string(source), cfg, filename)
		results = append(results, result)
	} else {
		for _, file := range files {
			source, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", err)
				return 2
			}
			cfg, cerr := configFor(file, opts, ruleMap)
			if cerr != nil {
				fmt.Fprintf(stderr, "Error: %v\n", cerr)
				return 2
			}
			applyCLIConfig(cfg, opts)

			var result *eslint.LintResult
			if opts.fix || opts.fixDryRun {
				result = linter.LintAndFix(string(source), cfg, file)
				if opts.fix && result.Fixed && result.Output != string(source) {
					if err := os.WriteFile(file, []byte(result.Output), 0o644); err != nil {
						fmt.Fprintf(stderr, "Error: %v\n", err)
						return 2
					}
				}
			} else {
				result = linter.Lint(string(source), cfg, file)
			}
			// ESLint's CLI engine reports absolute file paths, so the port does
			// too (the formatter prints them verbatim).
			result.FilePath = file
			result.UsedDeprecatedRules = deprecatedRulesUsed(cfg, ruleMap)
			results = append(results, result)
		}
	}

	if opts.quiet {
		for _, r := range results {
			kept := r.Messages[:0]
			for _, m := range r.Messages {
				if m.Severity == 2 || m.Fatal {
					kept = append(kept, m)
				}
			}
			r.Messages = kept
			*r = *eslint.NewLintResult(r.FilePath, kept)
			r.Source, r.Output, r.Fixed = "", "", false
		}
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].FilePath < results[j].FilePath })

	colorLevel := 0
	if opts.color {
		colorLevel = 1
	} else if !opts.noColor && isTerminal(stdout) {
		colorLevel = 1
	}

	output, err := formatters.FormatResults(opts.format, results, colorLevel)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if output != "" {
		if opts.outputFile != "" {
			if err := os.WriteFile(opts.outputFile, []byte(output+"\n"), 0o644); err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", err)
				return 2
			}
		} else {
			fmt.Fprintln(stdout, output)
		}
	}

	errorCount, warningCount := 0, 0
	for _, r := range results {
		errorCount += r.ErrorCount
		warningCount += r.WarningCount
	}
	if errorCount > 0 {
		return 1
	}
	if opts.maxWarnings >= 0 && warningCount > opts.maxWarnings {
		fmt.Fprintf(stderr, "ESLint found too many warnings (maximum: %d).\n", opts.maxWarnings)
		return 1
	}
	return 0
}

// defaultIgnorePatterns mirrors ESLint's defaults (node_modules and dotfiles).
func defaultIgnorePatterns() []string {
	return []string{"**/node_modules/**", ".*", "**/.*"}
}

// configFor resolves the config for a linted file.
func configFor(file string, opts *options, ruleMap map[string]eslint.Rule) (*eslint.Config, error) {
	if opts.configPath != "" {
		return loadConfigFile(opts.configPath, ruleMap)
	}
	if opts.noEslintrc {
		return eslint.NewConfig(), nil
	}
	cfg, err := loadConfigForFile(file, ruleMap)
	if err != nil {
		return nil, err
	}
	// ignorePatterns from the config chain apply to pattern expansion; the CLI
	// handles that separately, so nothing to do with cfg here.
	return cfg, nil
}

// applyCLIConfig applies --env and --global on top of the file config.
func applyCLIConfig(cfg *eslint.Config, opts *options) {
	for _, env := range opts.envs {
		for _, e := range strings.Split(env, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				cfg.Env[e] = true
			}
		}
	}
	for _, g := range opts.globals {
		for _, name := range strings.Split(g, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				cfg.Globals[name] = true
			}
		}
	}
}

// deprecatedRulesUsed lists the deprecated built-in rules a config enables, in
// configuration order — eslint's usedDeprecatedRules.
func deprecatedRulesUsed(cfg *eslint.Config, ruleMap map[string]eslint.Rule) []eslint.DeprecatedRuleInfo {
	out := []eslint.DeprecatedRuleInfo{}
	seen := map[string]bool{}
	for _, er := range cfg.EnabledRules(ruleMap) {
		if seen[er.ID] {
			continue
		}
		seen[er.ID] = true
		if replacedBy, ok := eslint.DeprecatedRules[er.ID]; ok {
			out = append(out, eslint.DeprecatedRuleInfo{RuleID: er.ID, ReplacedBy: replacedBy})
		}
	}
	return out
}

// listRules prints the implemented rule ids.
func listRules(stdout io.Writer) int {
	ids := rules.IDs()
	fmt.Fprintf(stdout, "Implemented rules (%d):\n", len(ids))
	for _, id := range ids {
		marker := ""
		if _, ok := eslint.RecommendedRules[id]; ok {
			marker = " (eslint:recommended)"
		}
		fmt.Fprintf(stdout, "  %s%s\n", id, marker)
	}
	return 0
}

// displayPath renders a path the way ESLint does: relative to cwd when inside
// it, absolute otherwise. (ESLint's own CLI reports absolute paths; this helper
// is kept for callers that want the short form.)
func displayPath(file, cwd string) string {
	rel, err := filepath.Rel(cwd, file)
	if err != nil {
		return file
	}
	if strings.HasPrefix(rel, "..") {
		return file
	}
	return rel
}

var _ = displayPath

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func parseArgs(args []string) (*options, error) {
	opts := &options{maxWarnings: -1, format: "stylish"}
	need := func(i int, name string) (string, error) {
		if i+1 >= len(args) {
			return "", fmt.Errorf("option %s requires a value", name)
		}
		return args[i+1], nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// --flag=value form.
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			args = append(args[:i], append([]string{parts[0], parts[1]}, args[i+1:]...)...)
			arg = parts[0]
		}
		switch arg {
		case "-h", "--help":
			opts.help = true
		case "-v", "--version":
			opts.versionOut = true
		case "--list-rules":
			opts.listRules = true
		case "-c", "--config":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.configPath = v
			i++
		case "--no-eslintrc":
			opts.noEslintrc = true
		case "-f", "--format":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.format = v
			i++
		case "-o", "--output-file":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.outputFile = v
			i++
		case "--ext":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.ext = v
			i++
		case "--ignore-pattern":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.ignorePatterns = append(opts.ignorePatterns, v)
			i++
		case "--global":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.globals = append(opts.globals, v)
			i++
		case "--env":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.envs = append(opts.envs, v)
			i++
		case "--no-ignore":
			opts.noIgnore = true
		case "--fix":
			opts.fix = true
		case "--fix-dry-run":
			opts.fixDryRun = true
		case "--quiet":
			opts.quiet = true
		case "--color":
			opts.color = true
		case "--no-color":
			opts.noColor = true
		case "--stdin":
			opts.stdin = true
		case "--stdin-filename":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			opts.stdinFilename = v
			i++
		case "--max-warnings":
			v, err := need(i, arg)
			if err != nil {
				return nil, err
			}
			n, cerr := strconv.Atoi(v)
			if cerr != nil {
				return nil, fmt.Errorf("invalid --max-warnings value %q", v)
			}
			opts.maxWarnings = n
			i++
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return nil, fmt.Errorf("unknown option %q", arg)
			}
			opts.patterns = append(opts.patterns, arg)
		}
	}
	return opts, nil
}
