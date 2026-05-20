package client

import (
	"regexp"
	"strconv"
	"strings"
)

// parseFlag extracts a named flag value from a command line string.
// Handles both "--flag=value" and "--flag value" forms.
func parseFlag(cmdLine, flag string) string {
	idx := strings.Index(cmdLine, flag)
	if idx < 0 {
		return ""
	}
	rest := cmdLine[idx+len(flag):]

	// "--flag=value"
	if len(rest) > 0 && rest[0] == '=' {
		return nextToken(rest[1:])
	}

	// "--flag value" (separated by whitespace)
	trimmed := strings.TrimLeft(rest, " \t")
	if len(trimmed) == 0 || trimmed == rest {
		// No whitespace separator found — not a valid form
		return ""
	}
	return nextToken(trimmed)
}

// nextToken returns the first whitespace-delimited token from s,
// stripping surrounding quotes if present.
func nextToken(s string) string {
	if len(s) == 0 {
		return ""
	}
	// Handle quoted values
	if s[0] == '"' || s[0] == '\'' {
		quote := s[0]
		end := strings.IndexByte(s[1:], quote)
		if end >= 0 {
			return s[1 : end+1]
		}
	}
	// Unquoted: take until whitespace
	end := strings.IndexAny(s, " \t\n")
	if end < 0 {
		return s
	}
	return s[:end]
}

// parseFlagInt extracts a named flag as an integer.
func parseFlagInt(cmdLine, flag string) int {
	v := parseFlag(cmdLine, flag)
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

// rankCandidate assigns a priority rank to a detected process.
// Higher rank = more likely to be the real language server.
// Uses a tiered priority scheme rather than point accumulation.
func rankCandidate(p *processInfo) int {
	lower := strings.ToLower(p.CommandLine)
	rank := 0

	// Tier 1: has CSRF token — definitive LS indicator
	if p.CSRFToken != "" {
		rank += 100
	}
	// Tier 2: binary name contains "language_server"
	if strings.Contains(lower, "language_server") ||
		strings.Contains(lower, "language-server") {
		rank += 40
	}
	// Tier 3: references the protobuf service
	if strings.Contains(lower, "exa.language_server_pb") {
		rank += 30
	}
	// Tier 4: has extension server port
	if p.ExtensionServerPort > 0 {
		rank += 15
	}
	// Tier 5: mentions LSP
	if strings.Contains(lower, "lsp") {
		rank += 8
	}
	// Tier 6: mentions antigravity at all
	if strings.Contains(lower, "antigravity") {
		rank += 3
	}

	return rank
}

// deduplicateProcesses filters out duplicate or redundant processes belonging to the same tool instance.
// For example, if both a global hub process and a workspace-specific process are running for the same IDE,
// we group them together and prefer the workspace-specific one (or the one with the highest PID).
func deduplicateProcesses(procs []*processInfo) []*processInfo {
	if len(procs) <= 1 {
		return procs
	}

	// Group by tool instance signature
	groups := make(map[string][]*processInfo)
	for _, p := range procs {
		sig := getToolSignature(p.CommandLine)
		groups[sig] = append(groups[sig], p)
	}

	var deduped []*processInfo
	for _, groupProcs := range groups {
		best := selectBestProcess(groupProcs)
		if best != nil {
			deduped = append(deduped, best)
		}
	}

	return deduped
}

func getToolSignature(cmdLine string) string {
	lower := strings.ToLower(cmdLine)

	// 1. Tool Type (main vs ide)
	toolType := "main"
	if strings.Contains(lower, "subclient_type ide") ||
		strings.Contains(lower, "antigravity ide") ||
		strings.Contains(lower, "antigravity-ide") {
		toolType = "ide"
	}

	// 2. App Data Dir
	appDataDir := parseFlag(cmdLine, "--app_data_dir")
	if appDataDir == "" {
		appDataDir = "default"
	}

	// 3. IDE Name override
	ideName := parseFlag(cmdLine, "--override_ide_name")
	if ideName == "" {
		ideName = "default"
	}

	return strings.ToLower(toolType + "|" + appDataDir + "|" + ideName)
}

func selectBestProcess(procs []*processInfo) *processInfo {
	if len(procs) == 0 {
		return nil
	}
	if len(procs) == 1 {
		return procs[0]
	}

	// 1. Prefer processes with a workspace_id
	var withWorkspace []*processInfo
	for _, p := range procs {
		if strings.Contains(strings.ToLower(p.CommandLine), "workspace_id") {
			withWorkspace = append(withWorkspace, p)
		}
	}

	candidates := procs
	if len(withWorkspace) > 0 {
		candidates = withWorkspace
	}

	// 2. Select the candidate with the highest PID
	best := candidates[0]
	for _, p := range candidates[1:] {
		if p.PID > best.PID {
			best = p
		}
	}
	return best
}

var (
	uuidRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
	hexRegex  = regexp.MustCompile(`^[a-fA-F0-9]{16,128}$`)
	numRegex  = regexp.MustCompile(`^[0-9]+$`)
)

func isTransientFlagName(flagName string) bool {
	f := strings.ToLower(flagName)
	return strings.Contains(f, "token") ||
		strings.Contains(f, "csrf") ||
		strings.Contains(f, "port") ||
		strings.Contains(f, "pipe")
}

func isTransientValue(v string) bool {
	// 1. Check if it's a number (port, etc.)
	if numRegex.MatchString(v) {
		return true
	}
	// 2. Check if it's a UUID
	if uuidRegex.MatchString(v) {
		return true
	}
	// 3. Check if it's a long hex token (e.g. CSRF token)
	if hexRegex.MatchString(v) {
		return true
	}
	// 4. Check if it's a named pipe or temp path
	vLower := strings.ToLower(v)
	if strings.Contains(vLower, "pipe") || strings.Contains(vLower, `\\.\`) {
		return true
	}
	return false
}

// normalizeCommandLine strips transient variable flags and values from the command line
// to create a stable signature representing the tool/IDE instance.
func normalizeCommandLine(cmdLine string) string {
	fields := strings.Fields(cmdLine)
	for i := 0; i < len(fields); i++ {
		field := fields[i]

		// If it's in the form --flag=value
		if idx := strings.IndexByte(field, '='); idx > 0 && strings.HasPrefix(field, "-") {
			flag := field[:idx]
			val := field[idx+1:]
			// Strip surrounding quotes if any
			trimmedVal := strings.Trim(val, `"'`)
			if isTransientFlagName(flag) || isTransientValue(trimmedVal) {
				fields[i] = flag + "=placeholder"
			}
			continue
		}

		// If it's a value following a flag (i.e. the previous field started with "-")
		if i > 0 && strings.HasPrefix(fields[i-1], "-") && !strings.HasPrefix(field, "-") {
			flag := fields[i-1]
			// Strip surrounding quotes
			trimmedVal := strings.Trim(field, `"'`)
			if isTransientFlagName(flag) || isTransientValue(trimmedVal) {
				fields[i] = "placeholder"
			}
		}
	}

	// Convert to lowercase for uniform comparison, and rejoin
	return strings.ToLower(strings.Join(fields, " "))
}
