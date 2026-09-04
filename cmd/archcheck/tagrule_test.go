// Tag-location linter. Enforces the rule that struct-tag families
// live in exactly one package each:
//
//	gorm:"..."      → only in internal/repositories/**
//	json:"..."      → only in internal/controllers/**
//	koanf:"..."     → only in internal/configurations/**
//	wire.NewSet     → only in internal/di/**
//
// A json tag on a domain type, or a gorm tag on a DTO, fails
// `go test` before the PR is opened.
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

type tagRule struct {
	tag     string
	allowed string
}


type forbiddenImport struct {
	importPath string
	allowed    string
	reason     string
}


var (
	tagRules = []tagRule{
		{tag: "gorm", allowed: "internal/repositories/"},
		{tag: "json", allowed: "internal/controllers/"},
		{tag: "koanf", allowed: "internal/configurations/"},
	}
	forbiddenImports = []forbiddenImport{
		{
			importPath: "github.com/google/wire",
			allowed:    "internal/di/",
			reason:     "wire.NewSet must only appear in the composition root",
		},
	}
)


func TestTagLocation(t *testing.T) {
	for _, path := range goFiles(t) {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		rel := projectRel(t, path)
		for _, rule := range tagRules {
			if !strings.HasPrefix(rel, rule.allowed) {
				assertNoTag(t, f, rel, rule)
			}
		}
	}
}


func TestForbiddenImportLocations(t *testing.T) {
	for _, path := range goFiles(t) {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		rel := projectRel(t, path)
		for _, rule := range forbiddenImports {
			if strings.HasPrefix(rel, rule.allowed) {
				continue
			}
			for _, imp := range f.Imports {
				if strings.Trim(imp.Path.Value, `"`) == rule.importPath {
					t.Errorf("%s: %s (allowed only in %s)", rel, rule.reason, rule.allowed)
				}
			}
		}
	}
}


// goFiles returns the absolute path of every .go file under
// internal/ that the rules apply to. Skips:
//   - generated mocks (under internal/mocks/)
//   - generated wire code (internal/di/wire_gen.go)
//   - test files (the rules don't apply to _test.go)
func goFiles(t *testing.T) []string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	rootSlash := filepath.ToSlash(root) + "/"
	var out []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		rel := filepath.ToSlash(path)
		if !strings.HasPrefix(rel, rootSlash+"internal/") {
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		if strings.Contains(rel, "/mocks/") ||
			strings.HasSuffix(rel, "_test.go") ||
			strings.HasSuffix(rel, "/wire_gen.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	require.NoError(t, err)
	return out
}


// projectRel returns a forward-slash path relative to the project
// root, e.g. /Users/.../simple-gin/internal/domains/product/core.go
// becomes internal/domains/product/core.go. Cross-platform via
// filepath.ToSlash.
func projectRel(t *testing.T, absPath string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	rel, err := filepath.Rel(root, absPath)
	require.NoError(t, err)
	return filepath.ToSlash(rel)
}


func assertNoTag(t *testing.T, f *ast.File, rel string, rule tagRule) {
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
			if strings.HasPrefix(tagVal, rule.tag+`:"`) || strings.HasPrefix(tagVal, rule.tag+":") {
				t.Errorf("%s: %s struct tag is only allowed in %s", rel, rule.tag, rule.allowed)
			}
		}
		return true
	})
}
