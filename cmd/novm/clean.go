// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type cleanCmd struct {
	id   string
	name string
}

func (*cleanCmd) Name() string     { return "clean" }
func (*cleanCmd) Synopsis() string { return "Remove metadata for a specific novm instance" }
func (*cleanCmd) Usage() string {
	return `clean [-id <id>] [-name <name>]:
  Remove stale metadata and control socket for a novm instance.

Flags:
`
}

func (c *cleanCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.id, "id", "", "instance ID (PID)")
	f.StringVar(&c.name, "name", "", "instance name")
}

func (c *cleanCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	target := c.id
	if target == "" {
		target = c.name
	}
	if target == "" && f.NArg() > 0 {
		target = f.Arg(0)
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "Error: must specify -id or -name")
		return subcommands.ExitUsageError
	}

	if err := RemoveInstance(target); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Printf("Cleaned instance %s\n", target)
	return subcommands.ExitSuccess
}

type cleanallCmd struct{}

func (*cleanallCmd) Name() string     { return "cleanall" }
func (*cleanallCmd) Synopsis() string { return "Remove metadata for all dead novm instances" }
func (*cleanallCmd) Usage() string {
	return `cleanall:
  Remove metadata and control sockets for all dead novm instances.
`
}

func (*cleanallCmd) SetFlags(_ *flag.FlagSet) {}

func (*cleanallCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	cleaned, err := CleanAllInstances()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning instances: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Printf("Cleaned %d dead instance(s)\n", cleaned)
	return subcommands.ExitSuccess
}
