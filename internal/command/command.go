package command

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/google/shlex"
)

// ShellProcess wraps a shell command execution
type ShellProcess struct {
	Command         string
	StripNewlines   bool
	ReturnErrOutput bool
	Timeout         int // in seconds
}

// NewShellProcess creates a new ShellProcess
func NewShellProcess(command string, timeout int) *ShellProcess {
	return &ShellProcess{
		Command:         command,
		StripNewlines:   false,
		ReturnErrOutput: true,
		Timeout:         timeout,
	}
}

// Run executes the command with the given arguments
func (s *ShellProcess) Run(args string) (string, error) {
	commands := args
	if args != "" && !strings.HasPrefix(commands, s.Command) {
		commands = s.Command + " " + commands
	} else if args == "" {
		commands = s.Command
	}

	return s.Exec(commands)
}

// Exec runs the commands and returns the output
func (s *ShellProcess) Exec(commands string) (string, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.Timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	// Parse the command string with proper handling of quotes
	parts, err := shlex.Split(commands)
	if err != nil {
		return "", err
	}

	if len(parts) > 1 {
		// Command with arguments
		// #nosec G204: Subprocess launched with a potential tainted input or cmd arguments
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	} else if len(parts) == 1 {
		// Single command without arguments
		// #nosec G204: Subprocess launched with a potential tainted input or cmd arguments
		cmd = exec.CommandContext(ctx, parts[0])
	} else {
		// Empty command
		return "", nil
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err = cmd.Run()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", ctx.Err()
	}

	// Handle errors
	if err != nil {
		if s.ReturnErrOutput && stderr.Len() > 0 {
			return stderr.String(), nil
		}
		return "", err
	}

	// Process output
	output := stdout.String()
	if s.StripNewlines {
		output = strings.TrimSpace(output)
	}

	return output, nil
}
