// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && amd64

package platform

import (
	"encoding/binary"
	"testing"
)

func TestNativeCpuidAsm(t *testing.T) {
	maxLeaf, ebx, ecx, edx := nativeCpuidAsm(0)
	if maxLeaf == 0 {
		t.Fatalf("nativeCpuidAsm(0) returned maxLeaf == 0")
	}

	// Manufacturer ID is EBX + EDX + ECX
	var vendor [12]byte
	binary.LittleEndian.PutUint32(vendor[0:4], ebx)
	binary.LittleEndian.PutUint32(vendor[4:8], edx)
	binary.LittleEndian.PutUint32(vendor[8:12], ecx)

	vendorStr := string(vendor[:])
	t.Logf("CPUID maxLeaf: %d, vendor: %s", maxLeaf, vendorStr)
	if vendorStr != "GenuineIntel" && vendorStr != "AuthenticAMD" {
		t.Logf("Notice: non-standard CPUID vendor string %q", vendorStr)
	}

	// Also verify function 1
	eax1, _, _, _ := nativeCpuidAsm(1)
	if eax1 == 0 {
		t.Errorf("nativeCpuidAsm(1) returned 0 for family/model/stepping")
	}
}
