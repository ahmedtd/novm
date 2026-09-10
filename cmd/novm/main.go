// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&createCmd{}, "")
	subcommands.Register(&runCmd{}, "")
	subcommands.Register(&listCmd{}, "")
	subcommands.Register(&cleanCmd{}, "")
	subcommands.Register(&cleanallCmd{}, "")
	subcommands.Register(&controlCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
