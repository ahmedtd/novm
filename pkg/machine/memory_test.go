// Copyright 2014 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package machine

import (
	"testing"

	"github.com/ahmedtd/novm/pkg/platform"
)

func TestMemoryRegionOverlaps(t *testing.T) {
	region := &MemoryRegion{
		Start: platform.Paddr(0x1000),
		Size:  0x1000, // [0x1000, 0x2000)
	}

	if region.End() != platform.Paddr(0x2000) {
		t.Errorf("End() = 0x%x, want 0x2000", region.End())
	}

	// Inside
	if !region.Contains(platform.Paddr(0x1100), 0x100) {
		t.Errorf("expected region to contain [0x1100, 0x1200)")
	}

	// Outside left
	if region.Contains(platform.Paddr(0x0f00), 0x100) {
		t.Errorf("expected region to not contain [0x0f00, 0x1000)")
	}

	// Outside right
	if region.Contains(platform.Paddr(0x2000), 0x100) {
		t.Errorf("expected region to not contain [0x2000, 0x2100)")
	}

	// Overlaps start
	if !region.Overlaps(platform.Paddr(0x0f00), 0x200) {
		t.Errorf("expected region to overlap [0x0f00, 0x1100)")
	}

	// Overlaps end
	if !region.Overlaps(platform.Paddr(0x1f00), 0x200) {
		t.Errorf("expected region to overlap [0x1f00, 0x2100)")
	}
}
