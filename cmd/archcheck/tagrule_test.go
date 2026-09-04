// Tag-location linter. Enforces the rule that struct-tag families
// live in exactly one package each:
//
//	gorm:"..."      → only in internal/repositories/**
//	json:"..."      → only in internal/controllers/**
//	koanf:"..."     → only in internal/configurations/**
//	wire.NewSet     → only in internal/di/**
//
// The original codebase had this as a convention enforced by
// code review. This analyzer turns it into a build-time rule:
// adding a json tag to a domain type, or a gorm tag to a DTO,
// breaks `go test` before the PR is opened.
package archcheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// tagRule: a struct-tag family allowed in exactly one path prefix.
type tagRule struct {
	tag     string // e.g. `gorm:"..."`
	allowed string // path prefix, e.g. `internal/repositories/`
}


var tagRules = []tagRule{
	{tag: "gorm", allowed: "internal/repositories/"},
	{tag: "json", allowed: "internal/controllers/"},
	{tag: "koanf", allowed: "internal/configurations/"},
}


// forbiddenImport: a package path that is allowed in exactly one place.
type forbiddenImport struct {
	importPath string // the import (e.g. "github.com/google/wire")
	allowed    string // path prefix where it's allowed
	reason     string // for the error message
}


var forbiddenImports = []forbiddenImport{
	{
		importPath: "github.com/google/wire",
		allowed:    "internal/di/",
		reason:     "wire.NewSet must only appear in the composition root",
	},
}


func TestTagLocation(t *testing.T) {
	root := repoRoot(t)

	fset := token.NewFileSet()
	for _, rule := range tagRules {
		walk(t, root, func(path string) {
			// Skip generated + vendor noise.
			if strings.Contains(path, "/mocks/") || strings.HasSuffix(path, "_test.go") {
				return
			}

			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return
			}

			rel, _ := filepath.Rel(root, path)
			if !strings.HasPrefix(rel, rule.allowed) {
				assertNoTag(t, fset, f, rel, rule.tag, path)
			}
		})
	}
}


func TestForbiddenImportLocations(t *testing.T) {
	root := repoRoot(t)
	fset := token.NewFileSet()
	for _, rule := range forbiddenImports {
		walk(t, root, func(path string) {
			if strings.Contains(path, "/mocks/") || strings.HasSuffix(path, "_test.go") {
				return
			}

			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return
			}

			rel, _ := filepath.Rel(root, path)
			if strings.HasPrefix(rel, rule.allowed) {
				return
			}

			for _, imp := range f.Imports {
				ip := strings.Trim(imp.Path.Value, `"`)
				if ip == rule.importPath {
					t.Errorf("%s: %s (allowed only in %s)", rel, rule.reason, rule.allowed)
				}
			}
		})
	}
}


func assertNoTag(t *testing.T, fset *token.FileSet, f *ast.File, rel, tag, path string) {
	t.Helper()
	ast.Inspect(f, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok {
			return true
		}

		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}

			tagVal := strings.Trim(field.Tag.Value, "`")
			if strings.HasPrefix(tagVal, tag+`:")`) || strings.HasPrefix(tagVal, tag+":") {
				t.Errorf("%s: %s struct tag is only allowed in %s%s — found in %s",
					rel, tag, "internal/", allowedDir(tag), path)
			}
		}

		return true
	})
}


// allowedDir maps a rule back to its directory for the error message.
func allowedDir(tag string) string {
	for _, r := range tagRules {
		if r.tag == tag {
			return strings.TrimSuffix(strings.TrimPrefix(r.allowed, "internal/"), "/")
		}
	}

	return ""
}


func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}


func walk(t *testing.T, root string, visit func(path string)) {
	t.Helper()
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		visit(path)
		return nil
	})
	require.NoError(t, err)
}
