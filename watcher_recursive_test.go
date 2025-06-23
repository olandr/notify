// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Edited by in 2025 olandr.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build (darwin && !kqueue && cgo) || windows
// +build darwin,!kqueue,cgo windows

package notify

import (
	"fmt"
	"testing"
)

// noevent stripts test-case from expected event list, used when action is not
// expected to trigger any events.
func noevent(cas FileOperation) FileOperation {
	return FileOperation{Action: cas.Action}
}

func TestWatcherRecursiveRewatch(t *testing.T) {
	w := newWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := []FileOperation{
		create(w, "src/github.com/rjeczalik/file"),
		create(w, "src/github.com/rjeczalik/dir/"),
		noevent(create(w, "src/github.com/rjeczalik/fs/dir/")),
		noevent(create(w, "src/github.com/dir/")),
		noevent(write(w, "src/github.com/rjeczalik/file", []byte("XD"))),
		noevent(rename(w, "src/github.com/rjeczalik/fs/LICENSE", "src/LICENSE")),
	}

	w.Watch("src/github.com/rjeczalik", Create, false)
	w.ExpectAny(cases)

	cases = []FileOperation{
		create(w, "src/github.com/rjeczalik/fs/file"),
		create(w, "src/github.com/rjeczalik/fs/cmd/gotree/file"),
		create(w, "src/github.com/rjeczalik/fs/cmd/dir/"),
		create(w, "src/github.com/rjeczalik/fs/cmd/gotree/dir/"),
		noevent(write(w, "src/github.com/rjeczalik/fs/file", []byte("XD"))),
		noevent(create(w, "src/github.com/anotherdir/")),
	}

	w.Rewatch("src/github.com/rjeczalik", "src/github.com/rjeczalik", Create, Create, true)
	w.ExpectAny(cases)

	cases = []FileOperation{
		create(w, "src/github.com/rjeczalik/1"),
		create(w, "src/github.com/rjeczalik/2/"),
		noevent(create(w, "src/github.com/rjeczalik/fs/cmd/1")),
		noevent(create(w, "src/github.com/rjeczalik/fs/1/")),
		noevent(write(w, "src/github.com/rjeczalik/fs/file", []byte("XD"))),
	}

	w.Rewatch("", "src/github.com/rjeczalik", Create, Create, false)
	w.ExpectAny(cases)
}

// TODO(rjeczalik): move to watcher_test.go after #5
func TestIsDirCreateEvent(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := [...]FileOperation{
		// i=0
		create(w, "src/github.com/jszwec/"),
		// i=1
		create(w, "src/github.com/jszwec/gojunitxml/"),
		// i=2
		create(w, "src/github.com/jszwec/gojunitxml/README.md"),
		// i=3
		create(w, "src/github.com/jszwec/gojunitxml/LICENSE"),
		// i=4
		create(w, "src/github.com/jszwec/gojunitxml/cmd/"),
	}

	dirs := [...]bool{
		true,  // i=0
		true,  // i=1
		false, // i=2
		false, // i=3
		true,  // i=4
	}

	fn := func(i int, _ FileOperation, ei EventInfo) error {
		d, ok := ei.(isDirer)
		if !ok {
			return fmt.Errorf("received EventInfo does not implement isDirer")
		}
		switch ok, err := d.isDir(); {
		case err != nil:
			return err
		case ok != dirs[i]:
			return fmt.Errorf("want ok=%v; got %v", dirs[i], ok)
		default:
			return nil
		}
	}

	w.ExpectAnyFunc(cases[:], fn)
}
