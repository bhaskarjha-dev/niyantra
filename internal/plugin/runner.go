package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxPluginStdoutBytes = 1024 * 1024
	maxPluginStderrBytes = 64 * 1024
)

// Run executes a plugin's capture action via subprocess.
// The plugin receives a JSON request on stdin and must write a JSON response to stdout.
// Uses exec.CommandContext for timeout enforcement — the process is killed if it
// exceeds the manifest's timeout (default 30s).
func (p *Plugin) Run(ctx context.Context, logger *slog.Logger) (*CaptureResult, error) {
	timeout := time.Duration(p.Manifest.EffectiveTimeout()) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Determine how to invoke the entry point.
	// For scripts (.py, .sh, .js, etc.), we need an interpreter prefix.
	// For compiled binaries, we invoke directly.
	cmd, err := p.buildCommand(ctx)
	if err != nil {
		return nil, err
	}
	cmd.Dir = p.Dir
	cmd.Env = pluginEnvironment()

	// Set up stdin pipe for sending the request
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("plugin %s: stdin pipe: %w", p.Manifest.ID, err)
	}

	// Capture stdout and stderr
	stdoutBuf := &limitedBuffer{limit: maxPluginStdoutBytes}
	stderrBuf := &limitedBuffer{limit: maxPluginStderrBytes}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("plugin %s: start: %w", p.Manifest.ID, err)
	}

	// Write the capture request to stdin, then close to signal EOF
	request := CaptureRequest{
		Action: "capture",
		Config: p.Config,
	}
	go func() {
		defer stdin.Close()
		json.NewEncoder(stdin).Encode(request)
	}()

	// Wait for process completion
	waitErr := cmd.Wait()

	// Log stderr output if any (plugin diagnostic messages)
	if stderrBuf.Len() > 0 {
		// Truncate long stderr to avoid log spam
		stderr := p.redact(strings.TrimSpace(stderrBuf.String()))
		if len(stderr) > 500 {
			stderr = stderr[:500] + "..."
		}
		logger.Debug("Plugin stderr", "plugin", p.Manifest.ID, "stderr", stderr)
	}

	// Check for context timeout
	if ctx.Err() != nil {
		return nil, fmt.Errorf("plugin %s: timed out after %s", p.Manifest.ID, timeout)
	}

	// Check for process exit error
	if waitErr != nil {
		stderr := p.redact(strings.TrimSpace(stderrBuf.String()))
		if len(stderr) > 200 {
			stderr = stderr[:200] + "..."
		}
		return nil, fmt.Errorf("plugin %s: exited with error: %w (stderr: %s)", p.Manifest.ID, waitErr, stderr)
	}
	if stdoutBuf.truncated {
		return nil, fmt.Errorf("plugin %s: stdout exceeded %d bytes", p.Manifest.ID, maxPluginStdoutBytes)
	}

	// Parse JSON response from stdout
	output := strings.TrimSpace(stdoutBuf.String())
	if output == "" {
		return nil, fmt.Errorf("plugin %s: empty stdout (no JSON response)", p.Manifest.ID)
	}

	var result CaptureResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// Truncate raw output for error message
		preview := output
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		preview = p.redact(preview)
		return nil, fmt.Errorf("plugin %s: invalid JSON response: %w (raw: %s)", p.Manifest.ID, err, preview)
	}

	return &result, nil
}

// buildCommand creates the exec.Cmd for running this plugin.
// It auto-detects the interpreter based on file extension.
func (p *Plugin) buildCommand(ctx context.Context) (*exec.Cmd, error) {
	entry := p.EntryPath
	ext := strings.ToLower(filepath.Ext(entry))

	switch ext {
	case ".py":
		// Try python3 first, fall back to python
		if _, err := exec.LookPath("python3"); err == nil {
			return exec.CommandContext(ctx, "python3", entry), nil
		}
		return exec.CommandContext(ctx, "python", entry), nil
	case ".js":
		return exec.CommandContext(ctx, "node", entry), nil
	case ".ts":
		return nil, fmt.Errorf("plugin %s: TypeScript entry points are disabled by default; compile to JavaScript or a native executable", p.Manifest.ID)
	case ".sh":
		return exec.CommandContext(ctx, "bash", entry), nil
	case ".ps1":
		return exec.CommandContext(ctx, "pwsh", "-File", entry), nil
	case ".rb":
		return exec.CommandContext(ctx, "ruby", entry), nil
	default:
		// Assume compiled binary or has shebang
		return exec.CommandContext(ctx, entry), nil
	}
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.truncated = true
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	_, _ = b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) Len() int {
	return b.buf.Len()
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func pluginEnvironment() []string {
	allowed := []string{
		"HOME",
		"PATH",
		"PATHEXT",
		"SYSTEMROOT",
		"TEMP",
		"TMP",
		"USERPROFILE",
		"HOMEDRIVE",
		"HOMEPATH",
		"WINDIR",
	}
	var env []string
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		for _, allowedName := range allowed {
			if strings.EqualFold(name, allowedName) {
				env = append(env, item)
				break
			}
		}
	}
	return env
}

func (p *Plugin) redact(text string) string {
	for key, value := range p.Config {
		if !looksSecretConfigKey(key) || len(value) < 4 {
			continue
		}
		text = strings.ReplaceAll(text, value, "[REDACTED]")
	}
	return text
}

func looksSecretConfigKey(key string) bool {
	lk := strings.ToLower(key)
	for _, pattern := range []string{"api_key", "token", "secret", "password", "pat", "credential"} {
		if strings.Contains(lk, pattern) {
			return true
		}
	}
	return false
}
