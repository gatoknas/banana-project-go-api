package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func TestRelevantGoFiles(t *testing.T) {
	tt := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "single handler source",
			in:   []string{"internal/handlers/user.go"},
			want: []string{"internal/handlers/user.go"},
		},
		{
			name: "test file is skipped",
			in:   []string{"internal/handlers/user_test.go"},
			want: nil,
		},
		{
			name: "generated pb file is skipped",
			in:   []string{"internal/handlers/api.pb.go"},
			want: nil,
		},
		{
			name: "models are skipped",
			in:   []string{"internal/models/user.go"},
			want: nil,
		},
		{
			name: "cmd main is skipped",
			in:   []string{"cmd/api/main.go"},
			want: nil,
		},
		{
			name: "db files are skipped",
			in:   []string{"db/migrations/0001_init.up.sql"},
			want: nil,
		},
		{
			name: "non-internal package is skipped",
			in:   []string{"internal/auth/jwt.go", "pkg/lib/helper.go"},
			want: []string{"internal/auth/jwt.go"},
		},
		{
			name: "multiple handler sources",
			in:   []string{"internal/service/category.go", "internal/service/user.go"},
			want: []string{"internal/service/category.go", "internal/service/user.go"},
		},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			got := relevantGoFiles(tt.in)
			if !equalString(got, tt.want) {
				t.Errorf("relevantGoFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tt := []struct {
		name string
		path string
		want bool
	}{
		{name: "models", path: "internal/models/user.go", want: true},
		{name: "cmd", path: "cmd/api/main.go", want: true},
		{name: "cmd nested", path: "cmd/api/main.go", want: true},
		{name: "db", path: "db/migrations/0001.up.sql", want: true},
		{name: "vendor", path: "vendor/foo.go", want: true},
		{name: "docs generated", path: "docs/docs.go", want: true},
		{name: "pb.go", path: "internal/handlers/api.pb.go", want: true},
		{name: "handler not excluded", path: "internal/handlers/user.go", want: false},
		{name: "service not excluded", path: "internal/service/user.go", want: false},
		{name: "auth not excluded", path: "internal/auth/jwt.go", want: false},
		{name: "middleware not excluded", path: "internal/middleware/cors.go", want: false},
		{name: "root main", path: "main.go", want: false},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExcluded(tt.path); got != tt.want {
				t.Errorf("isExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSiblingTest(t *testing.T) {
	tt := []struct {
		name string
		path string
		want string
	}{
		{name: "top level", path: "internal/handlers/user.go", want: "internal/handlers/user_test.go"},
		{name: "nested", path: "internal/service/email.go", want: "internal/service/email_test.go"},
		{name: "multi word", path: "internal/repository/user_account.go", want: "internal/repository/user_account_test.go"},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			if got := siblingTest(tt.path); got != tt.want {
				t.Errorf("siblingTest(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestChangedPackagePaths(t *testing.T) {
	tt := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "sources and tests collapse to packages",
			in:   []string{"internal/handlers/user.go", "internal/handlers/user_test.go", "internal/handlers/hello.go"},
			want: []string{"internal/handlers"},
		},
		{
			name: "excluded paths are dropped",
			in:   []string{"internal/models/user.go", "cmd/api/main.go", "db/schema.sql", "internal/service/user.go"},
			want: []string{"internal/service"},
		},
		{
			name: "non-go files ignored",
			in:   []string{"README.md", ".gitignore", "internal/handlers/category.go"},
			want: []string{"internal/handlers"},
		},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			got := changedPackagePaths(tt.in)
			if !equalString(got, tt.want) {
				t.Errorf("changedPackagePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangedFilePaths(t *testing.T) {
	tt := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "changed source kept, test dropped",
			in:   []string{"internal/handlers/user.go", "internal/handlers/user_test.go"},
			want: []string{"internal/handlers/user.go"},
		},
		{
			name: "excluded paths dropped",
			in:   []string{"internal/models/user.go", "cmd/api/main.go", "docs/docs.go", "internal/service/user.go"},
			want: []string{"internal/service/user.go"},
		},
		{
			name: "duplicates collapse and sort",
			in:   []string{"internal/service/user.go", "internal/service/user.go", "internal/handlers/product.go"},
			want: []string{"internal/handlers/product.go", "internal/service/user.go"},
		},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			got := changedFilePaths(tt.in)
			if !equalString(got, tt.want) {
				t.Errorf("changedFilePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCoverageTargets(t *testing.T) {
	files := []string{"internal/handlers/user.go", "internal/service/email.go"}
	tt := []struct {
		name  string
		scope string
		want  []string
	}{
		{
			name:  "file scope is the default target shape",
			scope: scopeFile,
			want:  []string{"internal/handlers/user.go", "internal/service/email.go"},
		},
		{
			name:  "package scope collapses to directories",
			scope: scopePackage,
			want:  []string{"internal/handlers", "internal/service"},
		},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			got := coverageTargets(files, tt.scope)
			if !equalString(got, tt.want) {
				t.Errorf("coverageTargets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	tt := []struct {
		name    string
		args    []string
		want    flags
		wantErr bool
	}{
		{
			name: "staged mode",
			args: []string{"-staged"},
			want: flags{staged: true},
		},
		{
			name: "base and scope",
			args: []string{"-base", "origin/main", "-scope", "package"},
			want: flags{base: "origin/main", scope: scopePackage},
		},
		{
			name: "root override",
			args: []string{"-all", "-root", "/tmp/repo"},
			want: flags{all: true, root: "/tmp/repo"},
		},
		{
			name:    "missing value",
			args:    []string{"-base"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"-nope"},
			wantErr: true,
		},
		{
			name:    "invalid scope",
			args:    []string{"-scope", "module"},
			wantErr: true,
		},
	}
	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseFlags() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadAndGenerateConfig(t *testing.T) {
	tmp := t.TempDir()
	base := config{
		Profile:   "cover.out",
		Threshold: threshold{File: 0, Package: 0, Total: 0},
		Exclude:   exclude{Paths: []string{"^internal/models", "^cmd/", "^db/"}},
	}
	data, err := yaml.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, baseConfigName), data, 0o600); err != nil {
		t.Fatal(err)
	}

	gen, cleanup, err := generateConfig(tmp, []string{"internal/handlers", "internal/service"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	out, err := os.ReadFile(gen)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, pkg := range []string{"internal/handlers", "internal/service"} {
		if !strings.Contains(text, pkg) {
			t.Errorf("generated config missing override for %q:\n%s", pkg, text)
		}
	}
	if !strings.Contains(text, "threshold: 80") {
		t.Errorf("generated config missing 80 threshold:\n%s", text)
	}
	if !strings.Contains(text, "profile: cover.out") {
		t.Errorf("generated config missing profile:\n%s", text)
	}
}

func TestMissingTestFiles(t *testing.T) {
	tmp := t.TempDir()
	mk := func(rel string) {
		p := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package handlers\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mk("internal/handlers/existing.go")
	mk("internal/handlers/existing_test.go")
	mk("internal/handlers/missing.go")
	mk("internal/models/model.go")
	mk("cmd/api/main.go")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	files := []string{
		"internal/handlers/existing.go",
		"internal/handlers/existing_test.go",
		"internal/handlers/missing.go",
		"internal/models/model.go",
		"cmd/api/main.go",
	}
	missing := missingTestFiles(files)
	if len(missing) != 1 || missing[0].file != "internal/handlers/missing.go" {
		t.Errorf("missingTestFiles() = %v, want exactly [internal/handlers/missing.go]", missing)
	}
}

func equalString(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
