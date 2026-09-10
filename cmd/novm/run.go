// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/google/subcommands"
)

type runCmd struct {
	id       string
	name     string
	terminal bool
	cwd      string
}

func (*runCmd) Name() string     { return "run" }
func (*runCmd) Synopsis() string { return "Run a command inside a running novm guest" }
func (*runCmd) Usage() string {
	return `run [-id <id>] [-name <name>] [-terminal] [-cwd <dir>] <command> [args...]:
  Execute a process inside the specified novm guest via noguest.

Flags:
`
}

func (r *runCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&r.id, "id", "", "instance ID (PID)")
	f.StringVar(&r.name, "name", "", "instance name")
	f.BoolVar(&r.terminal, "terminal", false, "allocate terminal")
	f.StringVar(&r.cwd, "cwd", "/", "working directory")
}

func (r *runCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	target := r.id
	if target == "" {
		target = r.name
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "Error: must specify -id or -name")
		return subcommands.ExitUsageError
	}

	cmdArgs := f.Args()
	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: must specify a command to execute")
		return subcommands.ExitUsageError
	}

	meta, err := FindInstance(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding instance %q: %v\n", target, err)
		return subcommands.ExitFailure
	}

	sockPath := ControlSocketPath(meta.Pid)
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to control socket %q: %v\n", sockPath, err)
		return subcommands.ExitFailure
	}
	defer conn.Close()

	// Send NOVM RUN header.
	if _, err := conn.Write([]byte("NOVM RUN\n")); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		return subcommands.ExitFailure
	}

	// Send StartCommand.
	startCmd := map[string]interface{}{
		"command":     cmdArgs,
		"environment": os.Environ(),
		"terminal":    r.terminal,
		"cwd":         r.cwd,
	}

	enc := json.NewEncoder(conn)
	if err := enc.Encode(startCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending start command: %v\n", err)
		return subcommands.ExitFailure
	}

	dec := json.NewDecoder(bufio.NewReader(conn))

	// First object is start status (nil if ok, string if error).
	var startErr *string
	if err := dec.Decode(&startErr); err != nil {
		fmt.Fprintf(os.Stderr, "Error receiving start response: %v\n", err)
		return subcommands.ExitFailure
	}
	if startErr != nil {
		fmt.Fprintf(os.Stderr, "Error starting command in guest: %s\n", *startErr)
		return subcommands.ExitFailure
	}

	// Goroutine for stdin forwarding.
	stdinDone := make(chan struct{})
	go func() {
		defer close(stdinDone)
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				encoded := base64.StdEncoding.EncodeToString(buf[:n])
				if err := enc.Encode(encoded); err != nil {
					return
				}
			}
			if err != nil {
				// Send empty string to indicate EOF.
				_ = enc.Encode("")
				return
			}
		}
	}()

	// Read stream loop.
	exitCode := 0
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Connection error: %v\n", err)
			}
			break
		}

		if len(raw) == 0 || string(raw) == "null" {
			// EOF from server.
			break
		}

		// Check if string (output data).
		var strData string
		if err := json.Unmarshal(raw, &strData); err == nil {
			decoded, err := base64.StdEncoding.DecodeString(strData)
			if err == nil {
				os.Stdout.Write(decoded)
			}
			continue
		}

		// Check if int (exit code).
		var code int
		if err := json.Unmarshal(raw, &code); err == nil {
			exitCode = code
			continue
		}
	}

	if exitCode != 0 {
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
