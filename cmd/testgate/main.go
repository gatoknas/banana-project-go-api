package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

const (
	defaultBase         = "origin/main"
	coverageThreshold   = 80
	baseConfigName      = ".testcoverage.yml"
	generatedConfigName = ".testcoverage.generated.yml"
	coverageProfileName = "cover.out"
	scopeFile           = "file"
	scopePackage        = "package"
)

type flags struct {
	staged bool
	all    bool
	base   string
	root   string
	scope  string
}

type filePair struct {
	file string
	test string
}

type override struct {
	Threshold int    `yaml:"threshold"`
	Path      string `yaml:"path"`
}

type threshold struct {
	File    int `yaml:"file"`
	Package int `yaml:"package"`
	Total   int `yaml:"total"`
}

type exclude struct {
	Paths []string `yaml:"paths,omitempty"`
}

type config struct {
	Profile   string     `yaml:"profile"`
	Threshold threshold  `yaml:"threshold"`
	Override  []override `yaml:"override,omitempty"`
	Exclude   exclude    `yaml:"exclude"`
}

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		fmt.Fprintln(os.Stderr, usage())
		return 1
	}
	if cfg.base == "" {
		cfg.base = defaultBase
	}
	if cfg.scope == "" {
		cfg.scope = scopeFile
	}
	root := cfg.root
	if root == "" {
		root = "."
	}

	if _, err := os.Stat(filepath.Join(root, baseConfigName)); err != nil {
		fmt.Fprintln(os.Stderr, "gate: missing", filepath.Join(root, baseConfigName), "is required:", err)
		return 1
	}
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(os.Stderr, "gate: go not found in PATH")
		return 1
	}

	files, err := gitChangedFiles(cfg, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate: failed to list changed files:", err)
		return 1
	}

	missing := missingTestFiles(files)
	if len(missing) > 0 {
		fmt.Println("rule 1: test presence -- FAIL")
		for _, m := range missing {
			fmt.Printf("  - %s: expected sibling test %s\n", m.file, m.test)
		}
		fmt.Println("Overall: FAIL")
		return 1
	}
	fmt.Println("rule 1: test presence -- PASS")

	if cfg.staged {
		fmt.Println("rule 2: coverage -- SKIP (-staged mode)")
		fmt.Println("Overall: PASS")
		return 0
	}

	if len(files) == 0 {
		fmt.Println("rule 2: coverage -- SKIP (no changed code)")
		fmt.Println("Overall: PASS")
		return 0
	}

	targets := coverageTargets(files, cfg.scope)
	genCfg, cleanup, err := generateConfig(root, targets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate: failed to build coverage config:", err)
		return 1
	}
	defer cleanup()

	if err := runCommand(root, os.Stdout, os.Stderr, "go", "test", "./...", "-coverprofile="+coverageProfileName, "-covermode=atomic", "-coverpkg=./..."); err != nil {
		fmt.Fprintln(os.Stderr, "gate: tests failed:", err)
		return 1
	}

	if err := runCommand(root, os.Stdout, os.Stderr, "go", "tool", "go-test-coverage", "--config="+genCfg); err != nil {
		fmt.Fprintln(os.Stderr, "gate: coverage check failed:", err)
		return 1
	}

	fmt.Println("Overall: PASS")
	return 0
}

func usage() string {
	return `usage: testgate [flags]

flags:
  -staged       run rule 1 on staged files (pre-commit)
  -base         base ref for diff vs HEAD (default: origin/main)
  -all          audit all Go files in the repo
  -scope        coverage target: file (default) or package
  -root         repo root directory (default: .)`
}

func parseFlags(args []string) (flags, error) {
	var cfg flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-staged":
			cfg.staged = true
		case "-all":
			cfg.all = true
		case "-base", "-root", "-scope":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("missing value for %s", args[i])
			}
			flag := args[i]
			i++
			switch flag {
			case "-base":
				cfg.base = args[i]
			case "-root":
				cfg.root = args[i]
			case "-scope":
				cfg.scope = args[i]
			}
		case "-h", "-help", "--help":
			fmt.Print(usage())
			os.Exit(0)
		default:
			return cfg, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	if cfg.scope != "" && cfg.scope != scopeFile && cfg.scope != scopePackage {
		return cfg, fmt.Errorf("invalid -scope %q (want %s or %s)", cfg.scope, scopeFile, scopePackage)
	}
	return cfg, nil
}

func gitChangedFiles(cfg flags, root string) ([]string, error) {
	var args []string
	if cfg.staged {
		args = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMR"}
	} else if cfg.all {
		return allGoFiles(root)
	} else {
		args = []string{"diff", "--name-only", "--diff-filter=ACMR", cfg.base + "...HEAD"}
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.ToSlash(line))
		}
	}
	return files, nil
}

func allGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files, err
}

func skipDir(path string) bool {
	name := filepath.ToSlash(path)
	for _, s := range []string{".git", "vendor", "node_modules", ".kilo", ".vscode", ".agents", "docs"} {
		if name == s || strings.HasPrefix(name, s+"/") {
			return true
		}
	}
	return false
}

func relevantGoFiles(files []string) []string {
	var out []string
	for _, f := range files {
		if strings.HasPrefix(f, "internal/") && !isExcluded(f) && !strings.HasSuffix(f, "_test.go") && !strings.HasSuffix(f, ".pb.go") {
			out = append(out, f)
		}
	}
	return out
}

func isExcluded(path string) bool {
	p := filepath.ToSlash(path)
	for _, s := range []string{"internal/models/", "internal/repository/", "cmd/", "db/", "vendor/", "docs/"} {
		if strings.HasPrefix(p, s) {
			return true
		}
	}
	return strings.HasSuffix(p, ".pb.go")
}

func missingTestFiles(files []string) []filePair {
	var out []filePair
	for _, f := range relevantGoFiles(files) {
		test := siblingTest(f)
		if !fileExists(test) {
			out = append(out, filePair{file: f, test: test})
		}
	}
	return out
}

func siblingTest(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.ToSlash(filepath.Join(dir, name+"_test.go"))
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func changedPackagePaths(files []string) []string {
	set := map[string]bool{}
	for _, f := range files {
		if !strings.HasSuffix(f, ".go") || isExcluded(f) {
			continue
		}
		pkg := filepath.ToSlash(filepath.Dir(f))
		set[pkg] = true
	}
	var out []string
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func coverageTargets(files []string, scope string) []string {
	if scope == scopePackage {
		return changedPackagePaths(files)
	}
	return changedFilePaths(files)
}

func changedFilePaths(files []string) []string {
	set := map[string]bool{}
	for _, f := range files {
		if !strings.HasSuffix(f, ".go") || strings.HasSuffix(f, "_test.go") || isExcluded(f) {
			continue
		}
		set[filepath.ToSlash(f)] = true
	}
	var out []string
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func generateConfig(root string, targets []string) (string, func(), error) {
	base, err := loadConfig(filepath.Join(root, baseConfigName))
	if err != nil {
		return "", nil, err
	}
	base.Override = nil
	for _, target := range targets {
		base.Override = append(base.Override, override{Path: regexp.QuoteMeta(target), Threshold: coverageThreshold})
	}
	if base.Override == nil {
		base.Override = []override{}
	}
	data, err := yaml.Marshal(base)
	if err != nil {
		return "", nil, err
	}
	tmp := filepath.Join(root, generatedConfigName)
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return "", nil, err
	}
	return tmp, func() {
		os.Remove(tmp)
		os.Remove(filepath.Join(root, coverageProfileName))
	}, nil
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	if cfg.Profile == "" {
		cfg.Profile = coverageProfileName
	}
	return cfg, nil
}

func runCommand(root string, stdout, stderr *os.File, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
