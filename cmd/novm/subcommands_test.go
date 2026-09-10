// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"testing"
)

func TestSubcommandNames(t *testing.T) {
	expected := []string{"create", "run", "list", "clean", "cleanall", "control"}
	commands := map[string]bool{
		new(createCmd).Name():   true,
		new(runCmd).Name():      true,
		new(listCmd).Name():     true,
		new(cleanCmd).Name():    true,
		new(cleanallCmd).Name(): true,
		new(controlCmd).Name():  true,
	}

	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected subcommand %q to be defined", name)
		}
	}
}

func TestCreateCommandFlags(t *testing.T) {
	cmd := new(createCmd)
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	cmd.SetFlags(fs)

	args := []string{
		"-cpus=4",
		"-mem=2048",
		"-name=my-vm",
		"-vmlinux=/path/to/vmlinux",
		"-cmdline=console=ttyS0",
		"-disk=filename=disk.img,dev=/dev/vda",
	}

	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if cmd.cpus != 4 {
		t.Errorf("cpus = %d, want 4", cmd.cpus)
	}
	if cmd.mem != 2048 {
		t.Errorf("mem = %d, want 2048", cmd.mem)
	}
	if cmd.name != "my-vm" {
		t.Errorf("name = %q, want my-vm", cmd.name)
	}
	if cmd.vmlinux != "/path/to/vmlinux" {
		t.Errorf("vmlinux = %q, want /path/to/vmlinux", cmd.vmlinux)
	}
	if cmd.cmdline != "console=ttyS0" {
		t.Errorf("cmdline = %q, want console=ttyS0", cmd.cmdline)
	}
	if len(cmd.disks) != 1 || cmd.disks[0] != "filename=disk.img,dev=/dev/vda" {
		t.Errorf("disks = %v, want [filename=disk.img,dev=/dev/vda]", cmd.disks)
	}
}

func TestListCommandFlags(t *testing.T) {
	cmd := new(listCmd)
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	cmd.SetFlags(fs)

	args := []string{"-alive=true", "-json=true"}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	if !cmd.alive {
		t.Errorf("cmd.alive = %v, want true", cmd.alive)
	}
	if !cmd.asJSON {
		t.Errorf("cmd.asJSON = %v, want true", cmd.asJSON)
	}
}
