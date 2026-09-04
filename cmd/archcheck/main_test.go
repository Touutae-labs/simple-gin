// Architecture enforcement test. Runs go-arch-lint against the
// project, parses the JSON output, and fails on any violation.
// go-arch-lint always exits 0 on warnings, so without this wrapper
// the linter is purely informational.
//
// Any forbidden edge (controller → repository, domain → gin, etc.)
// fails the build, so adding a new DTO, service, or adapter without
// updating the arch file or respecting the import graph becomes a
// red CI, not a code-review note.
package archcheck_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// archWarning mirrors the fields go-arch-lint emits per violation.
type archWarning struct {
	ComponentName      string `json:"ComponentName"`
	FileRelativePath   string `json:"FileRelativePath"`
	ResolvedImportName string `json:"ResolvedImportName"`
	Reference          struct {
		File   string `json:"File"`
		Line   int    `json:"Line"`
		Offset int    `json:"Offset"`
	} `json:"Reference"`
}


func TestArchLint_NoViolations(t *testing.T) {
	bin, err := exec.LookPath("go-arch-lint")
	if err != nil {
		// Not on PATH; try the standard GOPATH install location.
		bin = filepath.Join(os.Getenv("HOME"), "go", "bin", "go-arch-lint")
		if _, statErr := os.Stat(bin); statErr != nil {
			// Skip rather than fail — `make install` puts it on
			// PATH, but a fresh `go test ./...` before `make
			// install` shouldn't be red.
			t.Skipf("go-arch-lint not installed; run `make install` to enable this check (looked at %s)", bin)
		}
	}

	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	cmd := exec.Command(bin, "check", "--project-path", projectRoot, "--output-type", "json")
	out, runErr := cmd.Output()
	// go-arch-lint exits non-zero on any warning, so a non-nil
	// runErr is expected when violations are present. We always
	// parse the JSON to see what the linter actually said.
	if runErr != nil && len(out) == 0 {
		t.Fatalf("go-arch-lint produced no output and failed: %v", runErr)
	}

	var payload struct {
		Payload struct {
			ArchHasWarnings       bool          `json:"ArchHasWarnings"`
			ArchWarningsDeps      []archWarning `json:"ArchWarningsDeps"`
			ArchWarningsNotMatched []archWarning `json:"ArchWarningsNotMatched"`
			ArchWarningsDeepScan  []archWarning `json:"ArchWarningsDeepScan"`
		} `json:"Payload"`
	}
	require.NoError(t, json.Unmarshal(out, &payload), "parse go-arch-lint JSON: %s", string(out))

	if !payload.Payload.ArchHasWarnings {
		return
	}

	var report []string
	report = appendLines(report, "dep", payload.Payload.ArchWarningsDeps)
	report = appendLines(report, "not-matched", payload.Payload.ArchWarningsNotMatched)
	report = appendLines(report, "deep-scan", payload.Payload.ArchWarningsDeepScan)

	t.Fatalf("architecture violations found:\n%s\nfix the import or update .go-arch-lint.yml",
		joinLines(report))
}

func appendLines(buf []string, group string, ws []archWarning) []string {
	for _, w := range ws {
		buf = append(buf, group+": "+w.ComponentName+" -> "+w.ResolvedImportName+
			"  ("+w.FileRelativePath+":"+strconv.Itoa(w.Reference.Line)+")")
	}
	return buf
}

func joinLines(ls []string) string {
	out := ""
	for _, l := range ls {
		out += l + "\n"
	}
	return out
}
