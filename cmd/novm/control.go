// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/google/subcommands"
)

type controlCmd struct {
	id   string
	name string
}

func (*controlCmd) Name() string     { return "control" }
func (*controlCmd) Synopsis() string { return "Send a control command to a running novm instance" }
func (*controlCmd) Usage() string {
	return `control [-id <id>] [-name <name>] <command> [key=value ...]:
  Send an RPC command to the novm hypervisor control socket.

Examples:
  novm control -id 12345 vcpu id=0 paused=true
  novm control -id 12345 pause
  novm control -id 12345 unpause
  novm control -id 12345 state
  novm control -id 12345 trace enable=true

Flags:
`
}

func (c *controlCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.id, "id", "", "instance ID (PID)")
	f.StringVar(&c.name, "name", "", "instance name")
}

func parseArgValue(v string) interface{} {
	if n, err := strconv.ParseInt(v, 0, 64); err == nil {
		return n
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	var js interface{}
	if err := json.Unmarshal([]byte(v), &js); err == nil {
		return js
	}
	return v
}

func (c *controlCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	target := c.id
	if target == "" {
		target = c.name
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "Error: must specify -id or -name")
		return subcommands.ExitUsageError
	}

	args := f.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: must specify command (e.g. state, vcpu, pause, unpause, trace)")
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

	// 1. Send the 9-byte header.
	if _, err := conn.Write([]byte("NOVM RPC\n")); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		return subcommands.ExitFailure
	}

	// 2. Parse arguments as key=value.
	cmdName := args[0]
	params := make(map[string]interface{})
	for _, kv := range args[1:] {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parseArgValue(parts[1])
		} else {
			params[parts[0]] = true
		}
	}

	// Format method name, e.g. "vcpu" -> "Rpc.Vcpu"
	method := "Rpc." + strings.ToUpper(cmdName[:1]) + cmdName[1:]

	req := map[string]interface{}{
		"method": method,
		"params": []interface{}{params},
		"id":     1,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding RPC request: %v\n", err)
		return subcommands.ExitFailure
	}

	if _, err := conn.Write(append(reqData, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending RPC request: %v\n", err)
		return subcommands.ExitFailure
	}

	// 3. Read response.
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "Error: connection closed by server")
		}
		return subcommands.ExitFailure
	}

	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  interface{}     `json:"error"`
		Id     interface{}     `json:"id"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding RPC response: %v\n", err)
		return subcommands.ExitFailure
	}

	if resp.Error != nil {
		fmt.Fprintf(os.Stderr, "RPC Error: %v\n", resp.Error)
		return subcommands.ExitFailure
	}

	// Pretty print the result.
	if len(resp.Result) > 0 && string(resp.Result) != "null" {
		var pretty map[string]interface{}
		if err := json.Unmarshal(resp.Result, &pretty); err == nil {
			out, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println(string(resp.Result))
		}
	} else {
		fmt.Println("OK")
	}

	return subcommands.ExitSuccess
}
