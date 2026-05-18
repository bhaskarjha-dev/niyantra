package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	repoRoot, err := os.Getwd()
	if err != nil {
		fail("resolve working directory: %v", err)
	}

	files := []string{
		"README.md",
		"CHANGELOG.md",
		filepath.Join("docs", "API_SPEC.md"),
		filepath.Join("docs", "SECURITY.md"),
		filepath.Join("docs", "TESTING.md"),
		filepath.Join("docs", "USER_GUIDE.md"),
	}

	toolCount, err := countTools(filepath.Join(repoRoot, "internal", "mcpserver", "mcpserver.go"))
	if err != nil {
		fail("count tools: %v", err)
	}

	expectedFragments := []string{
		fmt.Sprintf("**%d tools:**", toolCount),
		"Budget headroom against recurring subscriptions",
		"heuristic git attribution",
		"observed token analytics",
	}

	var problems []string
	for _, rel := range files {
		path := filepath.Join(repoRoot, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: read failed: %v", rel, err))
			continue
		}
		text := string(data)
		if strings.ContainsRune(text, '\uFFFD') ||
			strings.Contains(text, "â") ||
			strings.Contains(text, "Â") ||
			strings.Contains(text, "Ã") ||
			strings.Contains(text, "ðŸ") {
			problems = append(problems, fmt.Sprintf("%s: appears to contain mojibake or replacement characters", rel))
		}
	}

	readmeBytes, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		problems = append(problems, fmt.Sprintf("README.md: read failed: %v", err))
	} else {
		readme := string(readmeBytes)
		for _, fragment := range expectedFragments {
			if !strings.Contains(readme, fragment) {
				problems = append(problems, fmt.Sprintf("README.md: missing expected fragment %q", fragment))
			}
		}
	}

	staleClaims := map[string][]string{
		"README.md": {
			"anomaly status card",
			"Safe to Spend guardrail",
			"For non-local binds, combine it with `--allow-remote` and `--auth user:pass`",
		},
		filepath.Join("docs", "API_SPEC.md"): {
			"Projected spend at current burn rate",
			"real per-commit cost",
			"actual AI token consumption from Claude Code sessions",
			"full-fidelity backup remains available via `GET /api/backup`",
			`"fullBackupPath": "/api/backup"`,
			"start Niyantra with `--mcp-http --allow-remote --auth user:pass`",
		},
		filepath.Join("docs", "SECURITY.md"): {
			"plaintext in SQLite",
			"not encrypted at rest",
			"the database contains quota percentages, not credentials",
			"non-local HTTP MCP requires `--auth`",
			"or `/api/backup`",
		},
		filepath.Join("docs", "TESTING.md"): {
			"budget_forecast | Burn rate, projected spend, on-track",
		},
		filepath.Join("docs", "USER_GUIDE.md"): {
			"also use `--allow-remote` and `--auth user:pass`",
		},
		filepath.Join("docs", "VISION.md"): {
			"Use `GET /api/backup`",
		},
	}

	for rel, claims := range staleClaims {
		path := filepath.Join(repoRoot, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: read failed: %v", rel, err))
			continue
		}
		text := string(data)
		for _, claim := range claims {
			if strings.Contains(text, claim) {
				problems = append(problems, fmt.Sprintf("%s: stale claim %q still present", rel, claim))
			}
		}
	}

	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Fprintf(os.Stderr, "docs sanity: %s\n", problem)
		}
		os.Exit(1)
	}
}

func countTools(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`mcp\.AddTool\(`)
	return len(re.FindAll(data, -1)), nil
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
