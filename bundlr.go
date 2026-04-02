// bundlr - merge source files from a directory tree into a single file
// Usage: bundlr . -o bundle.py -ext .py -exclude venv -include 'handler_*'
// Usage: bundlr -c /path/to/bundlr.yaml . -o bundle.py
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Config ────────────────────────────────────────────────────────────────────

// Config mirrors the fields in .bundlr.yaml
type Config struct {
	Ext     []string `yaml:"ext"`
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

// loadConfig reads the YAML file at the given path.
// If path is empty, it returns an empty Config with no error.
func loadConfig(path string) (Config, error) {
	var cfg Config
	if path == "" {
		return cfg, nil // no config — all defaults come from CLI flags
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("cannot read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("cannot parse config %s: %w", path, err)
	}

	fmt.Printf("Config   : %s\n", path)
	return cfg, nil
}

// ── multiFlag ─────────────────────────────────────────────────────────────────

// multiFlag allows a flag to be specified multiple times: -exclude venv -exclude dist
type multiFlag struct {
	values []string
	isSet  bool // true once the user provides the flag at least once
}

func (m *multiFlag) String() string { return strings.Join(m.values, ",") }
func (m *multiFlag) Set(v string) error {
	m.isSet = true
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			m.values = append(m.values, part)
		}
	}
	return nil
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	// ── Flags (CLI wins over config) ──────────────────────────────────────────
	configPath := flag.String("c", "", "Path to YAML config file (optional)")
	outputFile := flag.String("o", "all_in_one.py", "Output file path")

	var extFlag multiFlag
	flag.Var(&extFlag, "ext",
		"File extension(s) to collect, comma-separated or repeated.\n\tDefault: output file suffix from -o\n\tExample: -ext .go,.ts  or  -ext .go -ext .ts")

	var includeFlag multiFlag
	flag.Var(&includeFlag, "include",
		"Relative path glob(s) to include, matched from src using '/' separators. Supports '**' across directories.\n\tExample: -include 'cmd/**/handler_*.go' -include '**/*_test.go'")

	var excludeFlag multiFlag
	flag.Var(&excludeFlag, "exclude",
		"Relative path glob(s) to exclude, matched from src using '/' separators. Supports '**' across directories.\n\tExample: -exclude vendor -exclude dist -exclude 'internal/**/generated/*.go'")

	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fatalf("%v", err)
	}

	sourceDir := "."
	switch flag.NArg() {
	case 0:
	case 1:
		sourceDir = flag.Arg(0)
	default:
		fatalf("expected at most one source directory argument, got %d", flag.NArg())
	}

	// ── Merge config + CLI (CLI wins) ─────────────────────────────────────────
	// ext: CLI overrides config; if neither set, infer from -o
	var exts []string
	if extFlag.isSet {
		exts = extFlag.values
	} else if len(cfg.Ext) > 0 {
		exts = cfg.Ext
	}

	// ── Resolve paths ─────────────────────────────────────────────────────────
	absSrc, err := filepath.Abs(sourceDir)
	if err != nil {
		fatalf("cannot resolve source dir: %v", err)
	}
	absOut, err := filepath.Abs(*outputFile)
	if err != nil {
		fatalf("cannot resolve output file: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		fatalf("cannot create output directory: %v", err)
	}

	if len(exts) == 0 {
		outExt := strings.ToLower(filepath.Ext(absOut))
		if outExt == "" {
			fatalf("no file extension specified: use -ext or give -o a filename with an extension")
		}
		exts = []string{outExt}
	}

	// include: CLI overrides config entirely when set
	var includes []string
	if includeFlag.isSet {
		includes = includeFlag.values
	} else {
		includes = cfg.Include
	}

	// exclude: merge config defaults + CLI additions
	// (config provides the baseline; CLI can add more on top)
	excludes := mergeUnique(cfg.Exclude, excludeFlag.values)

	// ── Resolve extensions ────────────────────────────────────────────────────
	extensions := map[string]bool{}
	for _, e := range exts {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extensions[strings.ToLower(e)] = true
	}

	// ── Print summary ─────────────────────────────────────────────────────────
	fmt.Printf("Scanning : %s\n", absSrc)
	fmt.Printf("Output   : %s\n", absOut)
	fmt.Printf("Ext      : %s\n", formatSet(extensions))
	if len(includes) > 0 {
		fmt.Printf("Include  : %s\n", strings.Join(includes, ", "))
	}
	if len(excludes) > 0 {
		fmt.Printf("Exclude  : %s\n", strings.Join(excludes, ", "))
	}
	fmt.Println()

	// ── Collect matching files ────────────────────────────────────────────────
	var files []string

	err = filepath.WalkDir(absSrc, func(entryPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the output file itself
		if entryPath == absOut {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// -exclude: applied to both dirs and files
		if matchesAny(entryPath, absSrc, excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Extension filter
		name := d.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !extensions[ext] {
			return nil
		}

		// -include: relative path must match at least one pattern
		if len(includes) > 0 && !matchesAny(entryPath, absSrc, includes) {
			return nil
		}

		files = append(files, entryPath)
		return nil
	})
	if err != nil {
		fatalf("walk error: %v", err)
	}

	sort.Strings(files)

	// ── Write output ──────────────────────────────────────────────────────────
	out, err := os.Create(absOut)
	if err != nil {
		fatalf("cannot create output file: %v", err)
	}
	defer out.Close()

	for _, path := range files {
		rel, _ := filepath.Rel(absSrc, path)
		fmt.Fprintf(out, "# ===== File: %s =====\n\n", rel)

		in, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			continue
		}
		_, copyErr := io.Copy(out, in)
		in.Close()
		if copyErr != nil {
			fmt.Fprintf(os.Stderr, "warning: error reading %s: %v\n", path, copyErr)
		}
		fmt.Fprintf(out, "\n\n")
	}

	fmt.Printf("Done! Exported %d file(s) → %s\n", len(files), absOut)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func matchesAny(absPath, absSrc string, patterns []string) bool {
	rel, _ := filepath.Rel(absSrc, absPath)
	rel = filepath.ToSlash(rel)
	for _, pat := range patterns {
		pat = filepath.ToSlash(strings.TrimSpace(pat))
		if pat == "" {
			continue
		}
		if matchPathPattern(pat, rel) {
			return true
		}
	}
	return false
}

func matchPathPattern(pattern, rel string) bool {
	return matchPathSegments(splitPath(pattern), splitPath(rel))
}

func splitPath(s string) []string {
	if s == "" || s == "." {
		return nil
	}
	return strings.Split(s, "/")
}

func matchPathSegments(patternParts, relParts []string) bool {
	if len(patternParts) == 0 {
		return len(relParts) == 0
	}

	if patternParts[0] == "**" {
		if matchPathSegments(patternParts[1:], relParts) {
			return true
		}
		if len(relParts) > 0 {
			return matchPathSegments(patternParts, relParts[1:])
		}
		return false
	}

	if len(relParts) == 0 {
		return false
	}

	ok, err := pathpkg.Match(patternParts[0], relParts[0])
	if err != nil || !ok {
		return false
	}
	return matchPathSegments(patternParts[1:], relParts[1:])
}

func formatSet(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// mergeUnique combines two slices, deduplicating entries.
func mergeUnique(base, extra []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, v := range append(base, extra...) {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
