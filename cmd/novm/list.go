// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/subcommands"
)

type listCmd struct {
	alive bool
	asJSON bool
}

func (*listCmd) Name() string     { return "list" }
func (*listCmd) Synopsis() string { return "List novm instances" }
func (*listCmd) Usage() string {
	return `list [flags]:
  List running and known novm instances.

Flags:
`
}

func (l *listCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&l.alive, "alive", false, "show only alive instances")
	f.BoolVar(&l.asJSON, "json", false, "output in JSON format")
}

func (l *listCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	instances, err := ListInstances(l.alive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing instances: %v\n", err)
		return subcommands.ExitFailure
	}

	if l.asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(instances); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "PID\tNAME\tCPUS\tMEMORY\tSTATUS\tSTARTED")
	for _, inst := range instances {
		status := "dead"
		if inst.Alive {
			status = "alive"
		}
		started := "-"
		if inst.Timestamp > 0 {
			started = time.Unix(int64(inst.Timestamp), 0).Format("2006-01-02 15:04:05")
		}
		name := inst.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%d MB\t%s\t%s\n",
			inst.Pid, name, inst.Cpus, inst.Memory, status, started)
	}
	w.Flush()

	return subcommands.ExitSuccess
}
