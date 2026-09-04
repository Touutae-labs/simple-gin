// Architecture enforcement test. Runs go-arch-lint against the
// project, parses the JSON output, and fails on any violation.
// go-arch-lint itself always exits 0 on warnings, so without this
// wrapper the linter is purely informational.
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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// archWarning mirrors the fields go-arch-lint emits per violation.
type archWarning struct {
	ComponentName     string `json:"ComponentName"`
	FileRelativePath  string `json:"FileRelativePath"`
	ResolvedImportName string `json:"ResolvedImportName"`
	Reference         struct {
		File   string `json:"File"`
		Line   int    `json:"Line"`
		Offset int    `json:"Offset"`
	} `json:"Reference"`
}


func TestArchLint_NoViolations(t *testing.T) {
	bin, err := exec.LookPath("go-arch-lint")
	if err != nil {
		bin = filepath.Join(os.Getenv("HOME"), "go", "bin", "go-arch-lint")
	}

	_, err = os.Stat(bin)
	require.NoError(t, err, "go-arch-lint not found at %s; install with: go install github.com/fe3dback/go-arch-lint@v1.14.0", bin)

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
			ArchHasWarnings    bool          `json:"ArchHasWarnings"`
			ArchWarningsDeps   []archWarning `json:"ArchWarningsDeps"`
			ArchWarningsNotMatched []archWarning `json:"ArchWarningsNotMatched"`
			ArchWarningsDeepScan   []archWarning `json:"ArchWarningsDeepScan"`
		} `json:"Payload"`
	}

	require.NoError(t, json.Unmarshal(out, &payload), "parse go-arch-lint JSON: %s", string(out))

	if !payload.Payload.ArchHasWarnings {
		return
	}


	var b strings.Builder
	report := func(group string, ws []archWarning) {
		for _, w := range ws {
			b.WriteString(group + ": " + w.ComponentName + " → " + w.ResolvedImportName +
				"  (" + w.FileRelativePath + ":" + itoa(w.Reference.Line) + ")\n")
		}
	}

	report("dep", payload.Payload.ArchWarningsDeps)
	report("not-matched", payload.Payload.ArchWarningsNotMatched)
	report("deep-scan", payload.Payload.ArchWarningsDeepScan)

	t.Fatalf("architecture violations found:\n%s\nfix the import or update .go-arch-lint.yml", b.String())
}


func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	neg := n < 0
	if neg {
		n = -n
	}

	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	if neg {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}
