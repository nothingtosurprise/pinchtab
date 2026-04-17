package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const MaxToolOutputChars = 2400

type PersistentShell struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu     sync.Mutex
	buffer string
	closed bool
}

func NewPersistentShell(workDir string) (*PersistentShell, error) {
	cmd := exec.Command("/bin/bash")
	cmd.Dir = workDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}

	ps := &PersistentShell{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}

	go ps.drainStderr()

	return ps, nil
}

func (ps *PersistentShell) drainStderr() {
	io.Copy(io.Discard, ps.stderr)
}

func (ps *PersistentShell) Run(command string, timeout time.Duration) (output string, exitCode int, err error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return "", -1, errors.New("shell is closed")
	}

	nonce := randomHex(8)
	marker := fmt.Sprintf("__BENCH_DONE__%s", nonce)
	markerRegex := regexp.MustCompile(fmt.Sprintf(`\n%s:(\d+)\r?\n`, regexp.QuoteMeta(marker)))

	wrapped := fmt.Sprintf("%s\nprintf '\\n%s:%%s\\n' \"$?\"\n", command, marker)
	if _, err := io.WriteString(ps.stdin, wrapped); err != nil {
		return "", -1, fmt.Errorf("write command: %w", err)
	}

	deadline := time.Now().Add(timeout)
	reader := bufio.NewReader(ps.stdout)

	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining < 10*time.Millisecond {
			remaining = 10 * time.Millisecond
		}

		done := make(chan struct{})
		var chunk []byte
		var readErr error

		go func() {
			buf := make([]byte, 4096)
			n, e := reader.Read(buf)
			chunk = buf[:n]
			readErr = e
			close(done)
		}()

		select {
		case <-done:
			if readErr != nil && readErr != io.EOF {
				return "", -1, fmt.Errorf("read stdout: %w", readErr)
			}
			ps.buffer += string(chunk)

			if m := markerRegex.FindStringSubmatchIndex(ps.buffer); m != nil {
				output = ps.buffer[:m[0]]
				output = strings.TrimPrefix(output, "\n")
				var code int
				fmt.Sscanf(ps.buffer[m[2]:m[3]], "%d", &code)
				ps.buffer = ps.buffer[m[1]:]
				return output, code, nil
			}
		case <-time.After(remaining):
			ps.closeLocked(true)
			return "", -1, fmt.Errorf("command timed out after %v: %s", timeout, command)
		}
	}

	ps.closeLocked(true)
	return "", -1, fmt.Errorf("command timed out after %v: %s", timeout, command)
}

func (ps *PersistentShell) closeLocked(force bool) {
	if ps.closed {
		return
	}
	ps.closed = true

	if force {
		ps.cmd.Process.Kill()
	} else {
		io.WriteString(ps.stdin, "exit\n")
	}
	ps.stdin.Close()
	ps.cmd.Wait()
}

func (ps *PersistentShell) Close(force bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.closeLocked(force)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func TrimToolOutput(text string) string {
	if len(text) <= MaxToolOutputChars {
		return text
	}
	return fmt.Sprintf("%s\n\n[output truncated: %d more chars]", text[:MaxToolOutputChars], len(text)-MaxToolOutputChars)
}

func FormatToolResult(command string, exitCode int, output string) string {
	return fmt.Sprintf("$ %s\n[exit_code=%d]\n%s", command, exitCode, TrimToolOutput(output))
}
