// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/ahmedtd/novm/pkg/platform"
	"github.com/google/subcommands"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(val string) error {
	*s = append(*s, val)
	return nil
}

type createCmd struct {
	name        string
	cpus        int
	mem         int
	vmlinux     string
	initrd      string
	setup       string
	sysmap      string
	cmdline     string
	realInit    bool
	com1        bool
	com2        bool
	nopci       bool
	disks       stringList
	nics        stringList
	reads       stringList
	writes      stringList
	nofork      bool
	debug       bool
	step        bool
	paused      bool
	trace       bool
	novmmPath   string
	noguestPath string
}

func (*createCmd) Name() string     { return "create" }
func (*createCmd) Synopsis() string { return "Create and launch a new novm instance" }
func (*createCmd) Usage() string {
	return `create [flags]:
  Create and launch a new novm lightweight virtual machine.

Flags:
`
}

func (c *createCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.name, "name", "", "instance name")
	f.IntVar(&c.cpus, "cpus", 1, "number of vcpus")
	f.IntVar(&c.mem, "mem", 1024, "memory size in MB")
	f.StringVar(&c.vmlinux, "vmlinux", "", "path to vmlinux kernel ELF binary")
	f.StringVar(&c.initrd, "initrd", "", "path to initial ramdisk")
	f.StringVar(&c.setup, "setup", "", "path to setup header")
	f.StringVar(&c.sysmap, "sysmap", "", "path to System.map")
	f.StringVar(&c.cmdline, "cmdline", "", "additional kernel command line parameters")
	f.BoolVar(&c.realInit, "init", false, "use real in-guest init")
	f.BoolVar(&c.com1, "com1", false, "enable COM1 UART")
	f.BoolVar(&c.com2, "com2", false, "enable COM2 UART")
	f.BoolVar(&c.nopci, "nopci", false, "disable PCI (use MMIO)")
	f.Var(&c.disks, "disk", "define block device (filename=...,dev=...)")
	f.Var(&c.nics, "nic", "define network device (tapname=...,mac=...)")
	f.Var(&c.reads, "read", "read-only filesystem mapping (host_path or vm_path=>host_path)")
	f.Var(&c.writes, "write", "writable filesystem mapping (host_path or vm_path=>host_path)")
	f.BoolVar(&c.nofork, "nofork", false, "run novmm in foreground")
	f.BoolVar(&c.debug, "debug", false, "enable device debugging")
	f.BoolVar(&c.step, "step", false, "single step instructions")
	f.BoolVar(&c.paused, "paused", false, "start VM paused")
	f.BoolVar(&c.trace, "trace", false, "enable kernel symbol tracing")
	f.StringVar(&c.novmmPath, "novmm", "", "path to novmm executable")
	f.StringVar(&c.noguestPath, "noguest", "", "path to noguest executable")
}

func createTap(name string) (*os.File, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	var ifr struct {
		name  [16]byte
		flags uint16
		_     [22]byte
	}
	copy(ifr.name[:], []byte(name))
	ifr.flags = 0x0002 | 0x1000 // IFF_TAP | IFF_NO_PI
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(0x400454ca), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		file.Close()
		return nil, errno
	}
	return file, nil
}

func randomMac() string {
	buf := make([]byte, 3)
	rand.Read(buf)
	return fmt.Sprintf("28:48:46:%02x:%02x:%02x", buf[0], buf[1], buf[2])
}

func (c *createCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	novmmBin := c.novmmPath
	if novmmBin == "" {
		var err error
		novmmBin, err = FindBinary("novmm")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	if c.cpus < 1 {
		c.cpus = 1
	}
	if c.mem < 16 {
		c.mem = 16
	}

	// ExtraFiles tracking. In the child process:
	// extraFiles[0] becomes FD 3
	// extraFiles[1] becomes FD 4, etc.
	var extraFiles []*os.File
	nextFd := func(file *os.File) int {
		extraFiles = append(extraFiles, file)
		return 3 + len(extraFiles) - 1
	}

	// 1. Create state file (will be FD 3).
	stateFile, err := os.CreateTemp("", "novm-state-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating state temp file: %v\n", err)
		return subcommands.ExitFailure
	}
	defer stateFile.Close()
	os.Remove(stateFile.Name())
	stateFd := nextFd(stateFile)

	// 2. Create control socket (will be FD 4).
	if err := os.MkdirAll(ControlDir(), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating control dir: %v\n", err)
		return subcommands.ExitFailure
	}
	tmpSockPath := filepath.Join(ControlDir(), fmt.Sprintf("tmp-%d-%d.ctrl", os.Getpid(), time.Now().UnixNano()))
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: tmpSockPath, Net: "unix"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating control socket: %v\n", err)
		return subcommands.ExitFailure
	}
	ctrlFile, err := ln.File()
	ln.Close() // ln.File() duplicates the underlying file descriptor
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting control socket file: %v\n", err)
		return subcommands.ExitFailure
	}
	defer ctrlFile.Close()
	controlFd := nextFd(ctrlFile)

	// 3. Create user memory file.
	memFile, err := os.CreateTemp("", "novm-mem-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user memory temp file: %v\n", err)
		return subcommands.ExitFailure
	}
	defer memFile.Close()
	os.Remove(memFile.Name())
	if err := memFile.Truncate(int64(c.mem) * 1024 * 1024); err != nil {
		fmt.Fprintf(os.Stderr, "Error sizing user memory: %v\n", err)
		return subcommands.ExitFailure
	}
	memFd := nextFd(memFile)

	// Build devices list and cmdline tokens.
	var devices []map[string]interface{}
	var cmdlineTokens []string

	// Basic devices.
	devices = append(devices, map[string]interface{}{
		"name":   "bios",
		"driver": "bios",
	})
	devices = append(devices, map[string]interface{}{
		"name":   "acpi",
		"driver": "acpi",
	})
	cmdlineTokens = append(cmdlineTokens, "intel_pstate=disable")

	devices = append(devices, map[string]interface{}{
		"name":   "apic",
		"driver": "apic",
	})
	devices = append(devices, map[string]interface{}{
		"name":   "pit",
		"driver": "pit",
	})
	devices = append(devices, map[string]interface{}{
		"name":   "rtc",
		"driver": "rtc",
	})

	if !c.nopci {
		devices = append(devices, map[string]interface{}{
			"name":   "pci-bus",
			"driver": "pci-bus",
		})
		cmdlineTokens = append(cmdlineTokens, "pci=conf1")
	}

	// User memory device.
	devices = append(devices, map[string]interface{}{
		"name":   "user-memory",
		"driver": "user-memory",
		"data": map[string]interface{}{
			"fd": memFd,
		},
	})

	// UART devices.
	if c.com1 {
		devices = append(devices, map[string]interface{}{
			"name":   "uart-com1",
			"driver": "uart",
			"data": map[string]interface{}{
				"base":      0x3f8,
				"interrupt": 4,
			},
		})
		cmdlineTokens = append(cmdlineTokens, "console=uart,io,0x3f8")
	}
	if c.com2 {
		devices = append(devices, map[string]interface{}{
			"name":   "uart-com2",
			"driver": "uart",
			"data": map[string]interface{}{
				"base":      0x2f8,
				"interrupt": 3,
			},
		})
		cmdlineTokens = append(cmdlineTokens, "console=uart,io,0x2f8")
	}

	// Virtio Console.
	consoleDriver := "virtio-pci-console"
	if c.nopci {
		consoleDriver = "virtio-mmio-console"
	}
	devices = append(devices, map[string]interface{}{
		"name":   "console",
		"driver": consoleDriver,
	})

	// Virtio Disks.
	diskDriver := "virtio-pci-block"
	if c.nopci {
		diskDriver = "virtio-mmio-block"
	}
	for i, diskSpec := range c.disks {
		devName := fmt.Sprintf("vd%c", 'a'+i)
		fileName := ""
		for _, part := range strings.Split(diskSpec, ",") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				switch kv[0] {
				case "filename":
					fileName = kv[1]
				case "dev":
					devName = kv[1]
				}
			} else if fileName == "" {
				fileName = kv[0]
			}
		}
		if fileName == "" {
			continue
		}
		df, err := os.OpenFile(fileName, os.O_RDWR, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening disk %q: %v\n", fileName, err)
			return subcommands.ExitFailure
		}
		defer df.Close()
		diskFd := nextFd(df)
		devices = append(devices, map[string]interface{}{
			"name":   fmt.Sprintf("disk-%d", i),
			"driver": diskDriver,
			"data": map[string]interface{}{
				"dev": devName,
				"fd":  diskFd,
			},
		})
	}

	// Virtio NICs.
	nicDriver := "virtio-pci-net"
	if c.nopci {
		nicDriver = "virtio-mmio-net"
	}
	for i, nicSpec := range c.nics {
		tapName := fmt.Sprintf("novm%d-%d", os.Getpid(), i)
		mac := randomMac()
		for _, part := range strings.Split(nicSpec, ",") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				switch kv[0] {
				case "tapname":
					tapName = kv[1]
				case "mac":
					mac = kv[1]
				}
			}
		}
		tapFile, err := createTap(tapName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating tap device %q: %v\n", tapName, err)
			return subcommands.ExitFailure
		}
		defer tapFile.Close()
		tapFd := nextFd(tapFile)
		devices = append(devices, map[string]interface{}{
			"name":   fmt.Sprintf("net-%d", i),
			"driver": nicDriver,
			"data": map[string]interface{}{
				"mac": mac,
				"fd":  tapFd,
			},
		})
	}

	// Virtio Filesystems (Plan 9).
	fsDriver := "virtio-pci-fs"
	if c.nopci {
		fsDriver = "virtio-mmio-fs"
	}

	readMap := make(map[string][]string)
	if len(c.reads) == 0 {
		readMap["/"] = []string{"/"}
	} else {
		for _, r := range c.reads {
			parts := strings.SplitN(r, "=>", 2)
			if len(parts) == 2 {
				readMap[parts[0]] = append(readMap[parts[0]], parts[1])
			} else {
				readMap["/"] = append(readMap["/"], parts[0])
			}
		}
	}

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("novm-fs-%d-*", os.Getpid()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir for fs: %v\n", err)
		return subcommands.ExitFailure
	}
	writeMap := map[string]string{
		"/": tempDir,
	}
	for _, w := range c.writes {
		parts := strings.SplitN(w, "=>", 2)
		if len(parts) == 2 {
			writeMap[parts[0]] = parts[1]
		} else {
			writeMap["/"] = parts[0]
		}
	}

	devices = append(devices, map[string]interface{}{
		"name":   "fs-root",
		"driver": fsDriver,
		"data": map[string]interface{}{
			"tag":   "root",
			"read":  readMap,
			"write": writeMap,
		},
	})

	// In-guest init (noguest) fs device.
	if !c.realInit {
		noguestBin := c.noguestPath
		if noguestBin == "" {
			noguestBin, _ = FindBinary("noguest")
		}
		if noguestBin != "" {
			devices = append(devices, map[string]interface{}{
				"name":   "fs-init",
				"driver": fsDriver,
				"data": map[string]interface{}{
					"tag": "init",
					"read": map[string][]string{
						"/init": {noguestBin},
					},
					"write": map[string]string{},
				},
			})
		}
	}

	// Add MMIO device command line if nopci is enabled.
	if c.nopci {
		for i := range devices {
			addr := 0xe0000000 + i*4096
			irq := 32 + i
			cmdlineTokens = append(cmdlineTokens, fmt.Sprintf("virtio-mmio.%d@0x%x:%d:%d", i, addr, irq, i))
		}
	}

	if c.cmdline != "" {
		cmdlineTokens = append(cmdlineTokens, c.cmdline)
	}

	// Create Vcpus.
	vcpus := make([]platform.VcpuInfo, c.cpus)
	for i := 0; i < c.cpus; i++ {
		id := uint(i)
		vcpus[i].Id = &id
	}

	// Write machine state JSON to stateFile.
	fullState := map[string]interface{}{
		"vcpus":   vcpus,
		"devices": devices,
	}
	if err := json.NewEncoder(stateFile).Encode(fullState); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing machine state: %v\n", err)
		return subcommands.ExitFailure
	}
	if _, err := stateFile.Seek(0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "Error seeking state file: %v\n", err)
		return subcommands.ExitFailure
	}

	// Build novmm arguments.
	args := []string{
		fmt.Sprintf("-statefd=%d", stateFd),
		fmt.Sprintf("-controlfd=%d", controlFd),
	}
	if c.vmlinux != "" {
		args = append(args, fmt.Sprintf("-vmlinux=%s", c.vmlinux))
	}
	if c.initrd != "" {
		args = append(args, fmt.Sprintf("-initrd=%s", c.initrd))
	}
	if c.setup != "" {
		args = append(args, fmt.Sprintf("-setup=%s", c.setup))
	}
	if c.sysmap != "" {
		args = append(args, fmt.Sprintf("-sysmap=%s", c.sysmap))
	}
	if len(cmdlineTokens) > 0 {
		args = append(args, fmt.Sprintf("-cmdline=%s", strings.Join(cmdlineTokens, " ")))
	}
	if c.realInit {
		args = append(args, "-init=true")
	}
	if c.debug {
		args = append(args, "-debug=true")
	}
	if c.step {
		args = append(args, "-step=true")
	}
	if c.paused {
		args = append(args, "-paused=true")
	}
	if c.trace {
		args = append(args, "-trace=true")
	}

	cmd := exec.Command(novmmBin, args...)
	cmd.ExtraFiles = extraFiles

	if c.nofork {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting novmm: %v\n", err)
			return subcommands.ExitFailure
		}
		os.Rename(tmpSockPath, ControlSocketPath(cmd.Process.Pid))
		meta := &InstanceMetadata{
			Pid:       cmd.Process.Pid,
			Name:      c.name,
			Cpus:      c.cpus,
			Memory:    c.mem,
			Kernel:    c.vmlinux,
			Timestamp: float64(time.Now().Unix()),
			Alive:     true,
		}
		SaveInstance(meta)
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "novmm exited with: %v\n", err)
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}

	// Fork into background.
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err == nil {
		cmd.Stdin = devNull
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting novmm: %v\n", err)
		return subcommands.ExitFailure
	}

	finalSockPath := ControlSocketPath(cmd.Process.Pid)
	os.Rename(tmpSockPath, finalSockPath)

	meta := &InstanceMetadata{
		Pid:       cmd.Process.Pid,
		Name:      c.name,
		Cpus:      c.cpus,
		Memory:    c.mem,
		Kernel:    c.vmlinux,
		Timestamp: float64(time.Now().Unix()),
		Alive:     true,
	}
	SaveInstance(meta)

	fmt.Printf("%d\n", cmd.Process.Pid)
	return subcommands.ExitSuccess
}
