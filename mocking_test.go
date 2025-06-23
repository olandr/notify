// File created by olandr (c) 2025.
// Contains code from Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FakeWatcherCalls is a helping struct that implements the Watcher interace, but keeps everything for testing purposes.
type FakeWatcherCalls []Call

func (s *FakeWatcherCalls) Close() (_ error) { return }

func (s *FakeWatcherCalls) Watch(p string, e Event, isrec bool) (_ error) {
	dbgprintf("%s: (*FakeWatcherCalls).Watch(%q, %v)", caller(), p, e)
	*s = append(*s, Call{F: FuncWatch, P: p, E: e})
	if isrec {
		dbgprintf("%s: (*FakeWatcherCalls).RecursiveWatch(%q, %v)", caller(), p, e)
		*s = append(*s, Call{F: FuncRecursiveWatch, P: p, E: e})
	}
	return
}

func (s *FakeWatcherCalls) Unwatch(p string, isrec bool) (_ error) {
	dbgprintf("%s: (*FakeWatcherCalls).Unwatch(%q)", caller(), p)
	*s = append(*s, Call{F: FuncUnwatch, P: p})
	if isrec {
		dbgprintf("%s: (*FakeWatcherCalls).RecursiveUnwatch(%q)", caller(), p)
		*s = append(*s, Call{F: FuncRecursiveUnwatch, P: p})
	}
	return
}

func (s *FakeWatcherCalls) Rewatch(oldPath, newPath string, olde, newe Event, isrec bool) (_ error) {
	dbgprintf("%s: (*FakeWatcherCalls).Rewatch(%q, %v, %v)", caller(), newPath, olde, newe)
	*s = append(*s, Call{F: FuncRewatch, P: newPath, E: olde, NE: newe})
	if isrec {
		dbgprintf("%s: (*FakeWatcherCalls).RecursiveRewatch(%q, %q, %v, %v)", caller(), oldPath, newPath, olde, newe)
		*s = append(*s, Call{F: FuncRecursiveRewatch, P: oldPath, NP: newPath, E: olde, NE: newe})
	}
	return
}

// MockWatcher is a mock for Watcher interface.
type MockWatcher struct {
	Watcher watcher
	C       chan EventInfo
	Timeout time.Duration

	t    *testing.T
	root string
}

func (w *MockWatcher) Close() error {
	defer os.RemoveAll(w.root)
	if err := w.watcher().Close(); err != nil {
		w.Fatalf("w.Watcher.Close()=%v", err)
	}
	return nil
}
func (w *MockWatcher) Watch(path string, e Event, isrec bool) error {
	if err := w.watcher().Watch(w.clean(path), e, isrec); err != nil {
		if isrec {
			w.Fatalf("RecursiveWatch(%s, %v)=%v", path, e, err)
		} else {
			w.Fatalf("Watch(%s, %v)=%v", path, e, err)
		}
	}
	return nil
}

func (w *MockWatcher) Unwatch(path string, isrec bool) error {
	if err := w.watcher().Unwatch(w.clean(path), isrec); err != nil {
		if isrec {
			w.Fatalf("RecursiveUnwatch(%s)=%v", path, err)
		} else {
			w.Fatalf("Unwatch(%s)=%v", path, err)
		}
	}
	return nil
}

func (w *MockWatcher) Rewatch(oldPath, newPath string, olde, newe Event, isrec bool) error {
	if err := w.watcher().Rewatch(w.clean(oldPath), w.clean(newPath), olde, newe, isrec); err != nil {
		if isrec {
			w.Fatalf("RecursiveRewatch(%s, %s, %v, %v)=%v", oldPath, newPath, olde, newe, err)
		} else {
			w.Fatalf("Rewatch(%s, %v, %v)=%v", newPath, olde, newe, err)
		}
	}
	return nil
}

func (w *MockWatcher) initwatcher(buffer int) {
	c := make(chan EventInfo, buffer)
	w.Watcher = newWatcher(c)
	w.C = c
}

func (w *MockWatcher) clean(path string) string {
	path, isrec, err := cleanpath(filepath.Join(w.root, path))
	if err != nil {
		w.Fatalf("cleanpath(%q)=%v", path, err)
	}
	if isrec {
		path = path + "..."
	}
	return path
}
func (w *MockWatcher) watcher() watcher {
	if w.Watcher == nil {
		w.initwatcher(512)
	}
	return w.Watcher
}

func (w *MockWatcher) c() chan EventInfo {
	if w.C == nil {
		w.initwatcher(512)
	}
	return w.C
}

func (w *MockWatcher) timeout() time.Duration {
	if w.Timeout != 0 {
		return w.Timeout
	}
	return timeout()
}

func (w *MockWatcher) Fatal(v interface{}) {
	w.t.Fatalf("%s: %v", caller(), v)
}

func (w *MockWatcher) Fatalf(format string, v ...interface{}) {
	w.t.Fatalf("%s: %s", caller(), fmt.Sprintf(format, v...))
}

func (w *MockWatcher) ExpectAnyFunc(cases []FileOperation, fn FileOperationFunc) {
	UpdateWait() // Wait some time before starting the test.
Test:
	for i, cas := range cases {
		dbgprintf("ExpectAny: i=%d\n", i)
		cas.Action()
		Sync()
		switch cas.Events {
		case nil:
			if ei := drainall(w.C); len(ei) != 0 {
				w.Fatalf("unexpected dangling events: %v (i=%d)", ei, i)
			}
		default:
			select {
			case ei := <-w.C:
				dbgprintf("received: path=%q, event=%v, sys=%v (i=%d)", ei.Path(),
					ei.Event(), ei.Sys(), i)
				for j, want := range cas.Events {
					if err := EqualEventInfo(want, ei); err != nil {
						dbgprint(err, j)
						continue
					}
					if fn != nil {
						if err := fn(i, cas, ei); err != nil {
							w.Fatalf("ExpectAnyFunc(%d, %v)=%v", i, ei, err)
						}
					}
					drainall(w.C) // TODO(rjeczalik): revisit
					continue Test
				}
				w.Fatalf("ExpectAny received an event which does not match any of "+
					"the expected ones (i=%d): want one of %v; got %v", i, cas.Events, ei)
			case <-time.After(w.timeout()):
				w.Fatalf("timed out after %v waiting for one of %v (i=%d)", w.timeout(),
					cas.Events, i)
			}
			drainall(w.C) // TODO(rjeczalik): revisit
		}
	}
}

func (w *MockWatcher) ExpectAny(cases []FileOperation) {
	w.ExpectAnyFunc(cases, nil)
}

func (w *MockWatcher) aggregate(ei []EventInfo, pf string) (evs map[string]Event) {
	evs = make(map[string]Event)
	for _, cas := range ei {
		p := cas.Path()
		if pf != "" {
			p = filepath.Join(pf, p)
		}
		evs[p] |= cas.Event()
	}
	return
}

func (w *MockWatcher) ExpectAllFunc(cases []FileOperation) {
	UpdateWait() // Wait some time before starting the test.
	for i, cas := range cases {
		exp := w.aggregate(cas.Events, w.root)
		dbgprintf("ExpectAll: i=%d\n", i)
		cas.Action()
		Sync()
		got := w.aggregate(drainall(w.C), "")
		for ep, ee := range exp {
			ge, ok := got[ep]
			if !ok {
				w.Fatalf("missing events for %q (%v)", ep, ee)
				continue
			}
			delete(got, ep)
			if err := HasEventInfo(ee, ge, ep); err != nil {
				w.Fatalf("ExpectAll received an event which does not match "+
					"the expected ones for %q: want %v; got %v", ep, ee, ge)
				continue
			}
		}
		if len(got) != 0 {
			w.Fatalf("ExpectAll received unexpected events: %v", got)
		}
	}
}

// ExpectAll requires all requested events to be send.
// It does not require events to be send in the same order or in the same
// chunks (e.g. NoteWrite and NoteExtend reported as independent events are
// treated the same as one NoteWrite|NoteExtend event).
func (w *MockWatcher) ExpectAll(cases []FileOperation) {
	w.ExpectAllFunc(cases)
}
