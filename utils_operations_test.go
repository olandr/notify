// File created by olandr (c) 2025.
// Contains code from Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"os"
	"path/filepath"
	"strings"
)

type Case struct {
	Call     Call
	Record   []Call
	Receiver Chans
	Event    FileOperation
}

type FileOperation struct {
	Action func()
	Events []EventInfo
}

type FileOperationFunc func(i int, cas FileOperation, ei EventInfo) error

func (cas FileOperation) String() string {
	s := make([]string, 0, len(cas.Events))
	for _, ei := range cas.Events {
		s = append(s, "Event("+ei.Event().String()+")@"+filepath.FromSlash(ei.Path()))
	}
	return strings.Join(s, ", ")
}

func create(w *MockWatcher, path string) FileOperation {
	return FileOperation{
		Action: func() {
			isdir, err := tmpcreate(w.root, filepath.FromSlash(path))
			if err != nil {
				w.Fatalf("tmpcreate(%q, %q)=%v", w.root, path, err)
			}
			if isdir {
				dbgprintf("[FS] os.Mkdir(%q)\n", path)
			} else {
				dbgprintf("[FS] os.Create(%q)\n", path)
			}
		},
		Events: []EventInfo{
			&Call{P: path, E: Create},
		},
	}
}

func remove(w *MockWatcher, path string) FileOperation {
	return FileOperation{
		Action: func() {
			if err := os.RemoveAll(filepath.Join(w.root, filepath.FromSlash(path))); err != nil {
				w.Fatal(err)
			}
			dbgprintf("[FS] os.Remove(%q)\n", path)
		},
		Events: []EventInfo{
			&Call{P: path, E: Remove},
		},
	}
}

func rename(w *MockWatcher, oldpath, newpath string) FileOperation {
	return FileOperation{
		Action: func() {
			err := os.Rename(filepath.Join(w.root, filepath.FromSlash(oldpath)),
				filepath.Join(w.root, filepath.FromSlash(newpath)))
			if err != nil {
				w.Fatal(err)
			}
			dbgprintf("[FS] os.Rename(%q, %q)\n", oldpath, newpath)
		},
		Events: []EventInfo{
			&Call{P: newpath, E: Rename},
		},
	}
}

func write(w *MockWatcher, path string, p []byte) FileOperation {
	return FileOperation{
		Action: func() {
			f, err := os.OpenFile(filepath.Join(w.root, filepath.FromSlash(path)),
				os.O_WRONLY, 0644)
			if err != nil {
				w.Fatalf("OpenFile(%q)=%v", path, err)
			}
			if _, err := f.Write(p); err != nil {
				w.Fatalf("Write(%q)=%v", path, err)
			}
			if err := nonil(f.Sync(), f.Close()); err != nil {
				w.Fatalf("Sync(%q)/Close(%q)=%v", path, path, err)
			}
			dbgprintf("[FS] Write(%q)\n", path)
		},
		Events: []EventInfo{
			&Call{P: path, E: Write},
		},
	}
}
