// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstanceLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("NOVM_ROOT", tempDir)
	t.Setenv("NOVM_INSTANCES", filepath.Join(tempDir, "instances"))

	meta := &InstanceMetadata{
		Pid:    99999999, // Should not be alive
		Name:   "test-vm",
		Cpus:   2,
		Memory: 512,
		Kernel: "/boot/vmlinuz",
		Ips:    []string{"192.168.100.2"},
	}

	// 1. Save instance
	if err := SaveInstance(meta); err != nil {
		t.Fatalf("SaveInstance failed: %v", err)
	}

	// 2. List instances
	all, err := ListInstances(false)
	if err != nil {
		t.Fatalf("ListInstances(false) failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(all))
	}
	if all[0].Name != "test-vm" || all[0].Cpus != 2 || all[0].Memory != 512 {
		t.Errorf("unexpected metadata: %+v", all[0])
	}
	if all[0].Alive {
		t.Errorf("expected PID %d to not be alive", meta.Pid)
	}

	// 3. Find instance by name and by PID
	foundByName, err := FindInstance("test-vm")
	if err != nil {
		t.Fatalf("FindInstance(test-vm) failed: %v", err)
	}
	if foundByName.Pid != meta.Pid {
		t.Errorf("FindInstance by name gave pid %d, want %d", foundByName.Pid, meta.Pid)
	}

	foundByPid, err := FindInstance("99999999")
	if err != nil {
		t.Fatalf("FindInstance(99999999) failed: %v", err)
	}
	if foundByPid.Name != "test-vm" {
		t.Errorf("FindInstance by pid gave name %s, want test-vm", foundByPid.Name)
	}

	// 4. Clean dead instances
	cleaned, err := CleanAllInstances()
	if err != nil {
		t.Fatalf("CleanAllInstances failed: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("expected 1 instance cleaned, got %d", cleaned)
	}

	// Verify it is gone
	remaining, err := ListInstances(false)
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 instances after clean, got %d", len(remaining))
	}
}

func TestControlPaths(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("NOVM_ROOT", tempDir)

	sock := ControlSocketPath(1234)
	expectedSock := filepath.Join(tempDir, "control", "1234.ctrl")
	if sock != expectedSock {
		t.Errorf("ControlSocketPath = %q, want %q", sock, expectedSock)
	}
}

func TestIsProcessAlive(t *testing.T) {
	myPid := os.Getpid()
	if !IsProcessAlive(myPid) {
		t.Errorf("expected current process PID %d to be alive", myPid)
	}

	if IsProcessAlive(-1) {
		t.Errorf("expected PID -1 to not be alive")
	}
}
