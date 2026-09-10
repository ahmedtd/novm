// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpc

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

func openPty() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	// unlockpt: TIOCSPTLCK = 0x40045431
	var unlock int = 0
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), 0x40045431, uintptr(unsafe.Pointer(&unlock)))
	if e != 0 {
		master.Close()
		return nil, nil, e
	}
	// ptsname: TIOCGPTN = 0x80045430
	var ptn uint32
	_, _, e = syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), 0x80045430, uintptr(unsafe.Pointer(&ptn)))
	if e != 0 {
		master.Close()
		return nil, nil, e
	}
	slavePath := fmt.Sprintf("/dev/pts/%d", ptn)
	slave, err := os.OpenFile(slavePath, syscall.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, nil, err
	}
	return master, slave, nil
}

type StartCommand struct {

	// The command to run.
	Command []string `json:"command"`

	// The working directory.
	Cwd string `json:"cwd"`

	// Allocate a terminal?
	Terminal bool `json:"terminal"`

	// The environment.
	Environment []string `json:"environment"`
}

type StartResult struct {

	// The resulting pid.
	Pid int `json:"pid"`
}

func (server *Server) Start(
	command *StartCommand,
	result *StartResult) error {

	// We need at least a command.
	if len(command.Command) == 0 {
		return syscall.EINVAL
	}

	// Lookup our binary name.
	var binary string
	_, err := os.Stat(command.Command[0])
	if err == nil {
		// Absolute path is okay.
		binary = command.Command[0]
	} else {
		// Check our environment.
		for _, keyval := range command.Environment {
			if strings.HasPrefix(keyval, "PATH=") && len(keyval) > 5 {
				dirpaths := strings.Split(keyval[5:], ":")
				for _, dirpath := range dirpaths {
					testpath := path.Join(dirpath, command.Command[0])
					_, err = os.Stat(testpath)
					if err == nil {
						binary = testpath
						break
					}
				}
			}
		}
	}

	// Did we find a binary?
	if binary == "" {
		return syscall.ENOENT
	}

	var input *os.File
	var output *os.File

	var stdin *os.File
	var stdout *os.File
	var stderr *os.File

	if command.Terminal {
		master, slave, err := openPty()
		if err != nil {
			result.Pid = -1
			return err
		}
		defer slave.Close()

		// Set our inputs.
		input = master
		output = master
		stdin = slave
		stdout = slave
		stderr = slave

	} else {
		// Allocate pipes.
		r1, w1, err := os.Pipe()
		if err != nil {
			result.Pid = -1
			return err
		}
		r2, w2, err := os.Pipe()
		if err != nil {
			r1.Close()
			w1.Close()
			result.Pid = -1
			return err
		}

		defer r1.Close()
		defer w2.Close()

		// Set our inputs.
		input = w1
		output = r2
		stdin = r1
		stdout = w2
		stderr = w2
	}

	// Start the process.
	proc_attr := &os.ProcAttr{
		Dir:   command.Cwd,
		Env:   command.Environment,
		Files: []*os.File{stdin, stdout, stderr},
		Sys: &syscall.SysProcAttr{
			Setsid:  true,
			Setctty: command.Terminal,
			Ctty:    0,
		},
	}
	proc, err := os.StartProcess(
		binary,
		command.Command,
		proc_attr)

	// Unable to start?
	if err != nil {
		input.Close()
		if input != output {
			output.Close()
		}
		result.Pid = -1
		return err
	}

	// Create our process.
	process := &Process{
		input:     input,
		output:    output,
		starttime: time.Now(),
		cond:      sync.NewCond(&sync.Mutex{}),
	}

	// Save the pid.
	result.Pid = proc.Pid

	server.mutex.Lock()
	old_process := server.active[result.Pid]
	server.active[result.Pid] = process
	server.mutex.Unlock()

	if old_process != nil {
		old_process.close()
	}

	go server.wait()
	return nil
}
