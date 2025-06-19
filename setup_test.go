// File created by olandr (c) 2025
// Contains code from Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func newWatcherTest(t *testing.T, tree string) *MockWatcher {
	root, err := randomtree("")
	if err != nil {
		t.Fatalf(`tmptree("", %q)=%v`, tree, err)
	}
	root, _, err = cleanpath(root)
	if err != nil {
		t.Fatalf(`cleanpath(%q)=%v`, root, err)
	}
	Sync()
	return &MockWatcher{
		t:    t,
		root: root,
	}
}

func NewWatcherTest(t *testing.T, tree string, events ...Event) *MockWatcher {
	w := newWatcherTest(t, tree)
	if len(events) == 0 {
		events = []Event{Create, Remove, Write, Rename}
	}
	if rw, ok := w.watcher().(recursiveWatcher); ok {
		if err := rw.RecursiveWatch(w.root, joinevents(events)); err != nil {
			t.Fatalf("RecursiveWatch(%q, All)=%v", w.root, err)
		}
	} else {
		fn := func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				if err := w.watcher().Watch(path, joinevents(events)); err != nil {
					return err
				}
			}
			return nil
		}
		if err := filepath.Walk(w.root, fn); err != nil {
			t.Fatalf("Walk(%q, fn)=%v", w.root, err)
		}
	}
	drainall(w.C)
	return w
}
