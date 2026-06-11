package main

import (
	"os"
	"testing"
)

// TestCommandHelp exercises every subcommand's -h usage path (which prints the
// command's Usage closure and returns flag.ErrHelp, handled as a clean exit).
func TestCommandHelp(t *testing.T) {
	hermeticEnv(t)

	// Usage output goes to stderr; redirect it to a pipe to keep test logs quiet.
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	// Drain the pipe so writes never block.
	done := make(chan struct{})
	go func() { _, _ = readAll(r); close(done) }()
	restore := func() { os.Stderr = oldErr; w.Close(); <-done }

	cmds := []string{
		"search", "get", "create", "update", "comment", "delete",
		"spaces", "tree", "children", "move", "labels", "generate-skill",
	}
	for _, c := range cmds {
		if code := run([]string{c, "-h"}); code != 0 {
			os.Stderr = oldErr
			t.Errorf("%s -h exit = %d, want 0", c, code)
			os.Stderr = w
		}
	}
	restore()
}

func readAll(r *os.File) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			return buf, nil
		}
	}
}
