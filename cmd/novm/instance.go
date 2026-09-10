// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type InstanceMetadata struct {
	Pid       int       `json:"pid"`
	Name      string    `json:"name,omitempty"`
	Cpus      int       `json:"cpus"`
	Memory    int       `json:"memory"` // in MB
	Kernel    string    `json:"kernel,omitempty"`
	Ips       []string  `json:"ips,omitempty"`
	Timestamp float64   `json:"timestamp"`
	Alive     bool      `json:"alive,omitempty"`
}

func NovmRoot() string {
	if root := os.Getenv("NOVM_ROOT"); root != "" {
		return root
	}
	home := os.Getenv("HOME")
	if home != "" {
		p := filepath.Join(home, ".novm")
		if err := os.MkdirAll(p, 0755); err == nil {
			return p
		}
	}
	fallback := filepath.Join(os.TempDir(), ".novm")
	_ = os.MkdirAll(fallback, 0755)
	return fallback
}

func InstancesDir() string {
	if dir := os.Getenv("NOVM_INSTANCES"); dir != "" {
		return dir
	}
	return filepath.Join(NovmRoot(), "instances")
}

func ControlDir() string {
	return filepath.Join(NovmRoot(), "control")
}

func ControlSocketPath(pid int) string {
	return filepath.Join(ControlDir(), fmt.Sprintf("%d.ctrl", pid))
}

func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Check /proc/<pid> on Linux.
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
		return true
	}
	// Fallback to sending signal 0.
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func SaveInstance(meta *InstanceMetadata) error {
	dir := InstancesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if meta.Timestamp == 0 {
		meta.Timestamp = float64(time.Now().Unix())
	}
	metaPath := filepath.Join(dir, fmt.Sprintf("%d.json", meta.Pid))
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, data, 0644)
}

func ListInstances(aliveOnly bool) ([]*InstanceMetadata, error) {
	dir := InstancesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []*InstanceMetadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var meta InstanceMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		meta.Alive = IsProcessAlive(meta.Pid)
		if aliveOnly && !meta.Alive {
			continue
		}
		results = append(results, &meta)
	}
	return results, nil
}

func FindInstance(idOrName string) (*InstanceMetadata, error) {
	instances, err := ListInstances(false)
	if err != nil {
		return nil, err
	}

	// Try matching by PID first.
	if pid, err := strconv.Atoi(idOrName); err == nil {
		for _, inst := range instances {
			if inst.Pid == pid {
				return inst, nil
			}
		}
	}

	// Match by name.
	for _, inst := range instances {
		if inst.Name == idOrName {
			return inst, nil
		}
	}

	return nil, fmt.Errorf("instance %q not found", idOrName)
}

func RemoveInstance(idOrName string) error {
	meta, err := FindInstance(idOrName)
	if err != nil {
		return err
	}

	metaPath := filepath.Join(InstancesDir(), fmt.Sprintf("%d.json", meta.Pid))
	_ = os.Remove(metaPath)

	sockPath := ControlSocketPath(meta.Pid)
	_ = os.Remove(sockPath)

	return nil
}

func CleanAllInstances() (int, error) {
	instances, err := ListInstances(false)
	if err != nil {
		return 0, err
	}

	cleaned := 0
	for _, inst := range instances {
		if !inst.Alive {
			if err := RemoveInstance(strconv.Itoa(inst.Pid)); err == nil {
				cleaned++
			}
		}
	}
	return cleaned, nil
}

func FindBinary(name string) (string, error) {
	// Check environment variable.
	envKey := "NOVM_" + strings.ToUpper(name)
	if p := os.Getenv(envKey); p != "" {
		if _, err := os.Stat(p); err == nil {
			return filepath.Abs(p)
		}
	}

	// Check relative to current executable.
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(exeDir, name),
			filepath.Join(exeDir, "..", "bin", name),
			filepath.Join(exeDir, "..", "lib", "novm", "libexec", name),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return filepath.Abs(c)
			}
		}
	}

	// Check current working directory.
	if _, err := os.Stat(name); err == nil {
		return filepath.Abs(name)
	}
	if _, err := os.Stat(filepath.Join("bin", name)); err == nil {
		return filepath.Abs(filepath.Join("bin", name))
	}

	// Check PATH.
	if path, err := exec.LookPath(name); err == nil {
		return filepath.Abs(path)
	}

	return "", fmt.Errorf("binary %q not found (set %s or place in PATH/bin)", name, envKey)
}
