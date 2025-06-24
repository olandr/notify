// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Edited by in 2025 olandr.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// NOTE(rjeczalik): some useful environment variables:
//
//   - NOTIFY_DEBUG gives some extra information about generated events
//   - NOTIFY_TIMEOUT allows for changing default wait time for watcher's
//     events
//   - NOTIFY_TMP allows for changing location of temporary directory trees
//     created for test purpose

// FuncType represents enums for Watcher interface.
type FuncType string

const (
	FuncWatch            = FuncType("Watch")
	FuncUnwatch          = FuncType("Unwatch")
	FuncRewatch          = FuncType("Rewatch")
	FuncRecursiveWatch   = FuncType("RecursiveWatch")
	FuncRecursiveUnwatch = FuncType("RecursiveUnwatch")
	FuncRecursiveRewatch = FuncType("RecursiveRewatch")
	FuncStop             = FuncType("Stop")
)

type Chans []chan EventInfo

func NewChans(n int) Chans {
	ch := make([]chan EventInfo, n)
	for i := range ch {
		ch[i] = make(chan EventInfo, buffer)
	}
	return ch
}

func (c Chans) Foreach(fn func(chan<- EventInfo, node)) {
	for i, ch := range c {
		fn(ch, node{Name: strconv.Itoa(i)})
	}
}

func (c Chans) Drain() (ei []EventInfo) {
	n := len(c)
	stop := make(chan struct{})
	eich := make(chan EventInfo, n*buffer)
	go func() {
		defer close(eich)
		cases := make([]reflect.SelectCase, n+1)
		for i := range c {
			cases[i].Chan = reflect.ValueOf(c[i])
			cases[i].Dir = reflect.SelectRecv
		}
		cases[n].Chan = reflect.ValueOf(stop)
		cases[n].Dir = reflect.SelectRecv
		for {
			i, v, ok := reflect.Select(cases)
			if i == n {
				return
			}
			if !ok {
				panic("(Chans).Drain(): unexpected chan close")
			}
			eich <- v.Interface().(EventInfo)
		}
	}()
	<-time.After(50 * time.Duration(n) * time.Millisecond)
	close(stop)
	for e := range eich {
		ei = append(ei, e)
	}
	return
}

// Call represents single call to Watcher issued by the tree and is recorded by a FakeWatcherCall.
type Call struct {
	F   FuncType       // denotes type of function to call, for both watcher and notifier interface
	C   chan EventInfo // user channel being an argument to either Watch or Stop function
	P   string         // regular Path argument and old path from RecursiveRewatch call
	NP  string         // new Path argument from RecursiveRewatch call
	E   Event          // regular Event argument and old Event from a Rewatch call
	NE  Event          // new Event argument from Rewatch call
	S   interface{}    // when Call is used as EventInfo, S is a value of Sys()
	Dir bool           // when Call is used as EventInfo, Dir is a value of isDir()
}

// Call implements the EventInfo interface.
func (c *Call) Event() Event         { return c.E }
func (c *Call) Path() string         { return c.P }
func (c *Call) String() string       { return fmt.Sprintf("%#v", c) }
func (c *Call) Sys() interface{}     { return c.S }
func (c *Call) isDir() (bool, error) { return c.Dir, nil }

// CallSlice is a convenient wrapper for a slice of Call values, which allows
// to sort them in ascending order.
type CallSlice []Call

// CallSlice implements sort.Interface inteface.
func (cs CallSlice) Len() int           { return len(cs) }
func (cs CallSlice) Less(i, j int) bool { return cs[i].P < cs[j].P }
func (cs CallSlice) Swap(i, j int)      { cs[i], cs[j] = cs[j], cs[i] }
func (cs CallSlice) Sort()              { sort.Sort(cs) }

type N struct {
	Timeout time.Duration

	t        *testing.T
	tree     tree
	w        *MockWatcher
	spy      *FakeWatcherCalls
	c        chan EventInfo
	j        int // spy offset
	realroot string
}

func newN(t *testing.T, tree string) *N {
	n := &N{
		t: t,
		w: newWatcherTest(t, tree),
	}
	realroot, err := canonical(n.w.root)
	if err != nil {
		t.Fatalf("%s: unexpected fixture failure: %v", caller(), err)
	}
	n.realroot = realroot
	return n
}

func newTreeN(t *testing.T, tree string) *N {
	c := make(chan EventInfo, buffer)
	n := newN(t, tree)
	n.spy = &FakeWatcherCalls{}
	n.w.Watcher = n.spy
	n.w.C = c
	n.c = c
	return n
}

func NewNotifyTest(t *testing.T, tree string) *N {
	n := newN(t, tree)
	n.tree = NewTree()
	t.Cleanup(n.Close)
	return n
}

func (n *N) timeout() time.Duration {
	if n.Timeout != 0 {
		return n.Timeout
	}
	return n.w.timeout()
}

func (n *N) W() *MockWatcher {
	return n.w
}

func (n *N) Close() {
	err := n.tree.Close()
	os.RemoveAll(n.w.root)
	if err != nil {
		n.w.Fatalf("(notifier).Close()=%v", err)
	}
}

func dummyDoNotWatch(path string) bool {
	return false
}

func (n *N) Watch(path string, c chan<- EventInfo, events ...Event) {
	UpdateWait() // we need to wait on Windows because of its asynchronous watcher.
	path = filepath.Join(n.w.root, path)
	if err := n.tree.Watch(path, c, dummyDoNotWatch, events...); err != nil {
		n.t.Errorf("Watch(%s, %p, %v)=%v", path, c, events, err)
	}
}

func (n *N) WatchErr(path string, c chan<- EventInfo, err error, events ...Event) {
	path = filepath.Join(n.w.root, path)
	switch e := n.tree.Watch(path, c, dummyDoNotWatch, events...); {
	case err == nil && e == nil:
		n.t.Errorf("Watch(%s, %p, %v)=nil", path, c, events)
	case err != nil && e != err:
		n.t.Errorf("Watch(%s, %p, %v)=%v != %v", path, c, events, e, err)
	}
}

func (n *N) Stop(c chan<- EventInfo) {
	n.tree.Stop(c)
}

func (n *N) Call(calls ...Call) {
	for i := range calls {
		switch calls[i].F {
		case FuncWatch:
			n.Watch(calls[i].P, calls[i].C, calls[i].E)
		case FuncStop:
			n.Stop(calls[i].C)
		default:
			panic("unsupported call type: " + string(calls[i].F))
		}
	}
}

func (n *N) expectDry(ch Chans, i int) {
	if ei := ch.Drain(); len(ei) != 0 {
		n.w.Fatalf("unexpected dangling events: %v (i=%d)", ei, i)
	}
}

func (n *N) ExpectRecordedCalls(cases []Case) {
	for i, cas := range cases {
		dbgprintf("ExpectRecordedCalls: i=%d\n", i)
		n.Call(cas.Call)
		record := (*n.spy)[n.j:]
		if len(cas.Record) == 0 && len(record) == 0 {
			continue
		}
		n.j = len(*n.spy)
		if len(record) != len(cas.Record) {
			n.t.Fatalf("%s: want len(record)=%d; got %d [%+v] (i=%d)", caller(),
				len(cas.Record), len(record), record, i)
		}
		CallSlice(record).Sort()
		for j := range cas.Record {
			if err := EqualCall(cas.Record[j], record[j]); err != nil {
				n.t.Fatalf("%s: %v (i=%d, j=%d)", caller(), err, i, j)
			}
		}
	}
}

func (n *N) collect(ch Chans) <-chan []EventInfo {
	done := make(chan []EventInfo)
	go func() {
		cases := make([]reflect.SelectCase, len(ch))
		unique := make(map[<-chan EventInfo]EventInfo, len(ch))
		for i := range ch {
			cases[i].Chan = reflect.ValueOf(ch[i])
			cases[i].Dir = reflect.SelectRecv
		}
		for i := len(cases); i != 0; i = len(cases) {
			j, v, ok := reflect.Select(cases)
			if !ok {
				n.t.Fatal("unexpected chan close")
			}
			ch := cases[j].Chan.Interface().(chan EventInfo)
			got := v.Interface().(EventInfo)
			if ei, ok := unique[ch]; ok {
				n.t.Fatalf("duplicated event %v (previous=%v) received on collect", got, ei)
			}
			unique[ch] = got
			cases[j], cases = cases[i-1], cases[:i-1]
		}
		collected := make([]EventInfo, 0, len(ch))
		for _, ch := range unique {
			collected = append(collected, ch)
		}
		done <- collected
	}()
	return done
}

func (n *N) abs(rel Call) *Call {
	rel.P = filepath.Join(n.realroot, filepath.FromSlash(rel.P))
	if !filepath.IsAbs(rel.P) {
		rel.P = filepath.Join(wd, rel.P)
	}
	return &rel
}

func (n *N) ExpectTreeEvents(cases []Case, all Chans) {
	for i, cas := range cases {
		dbgprintf("ExpectTreeEvents: i=%d\n", i)
		// Ensure there're no dangling event left by previous test-case.
		n.expectDry(all, i)
		n.c <- n.abs(cas.Call)
		switch cas.Receiver {
		case nil:
			n.expectDry(all, i)
		default:
			ch := n.collect(cas.Receiver)
			select {
			case collected := <-ch:
				for _, got := range collected {
					if err := EqualEventInfo(&cas.Call, got); err != nil {
						n.w.Fatalf("%s: %s (i=%d)", caller(), err, i)
					}
				}
			case <-time.After(n.timeout()):
				n.w.Fatalf("ExpectTreeEvents has timed out after %v waiting for"+
					" %v on %s (i=%d)", n.timeout(), cas.Call.E, cas.Call.P, i)
			}

		}
	}
	n.expectDry(all, -1)
}

func (n *N) ExpectNotifyEvents(cases []Case, all Chans) {
	UpdateWait() // Wait some time before starting the test.
	for i, cas := range cases {
		dbgprintf("ExpectNotifyEvents: i=%d\n", i)
		cas.Event.Action()
		Sync()
		switch cas.Receiver {
		case nil:
			n.expectDry(all, i)
		default:
			ch := n.collect(cas.Receiver)
			select {
			case collected := <-ch:
			Compare:
				for j, ei := range collected {
					dbgprintf("received: path=%q, event=%v, sys=%v (i=%d, j=%d)", ei.Path(),
						ei.Event(), ei.Sys(), i, j)
					for _, want := range cas.Event.Events {
						if err := EqualEventInfo(want, ei); err != nil {
							dbgprint(err, j)
							continue
						}
						continue Compare
					}
					n.w.Fatalf("ExpectNotifyEvents received an event which does not"+
						" match any of the expected ones (i=%d): want one of %v; got %v", i,
						cas.Event.Events, ei)
				}
			case <-time.After(n.timeout()):
				n.w.Fatalf("ExpectNotifyEvents did not receive any of the expected events [%v] "+
					"after %v (i=%d)", cas.Event, n.timeout(), i)
			}
		}
	}
	n.expectDry(all, -1)
}

func (n *N) Walk(fn walkFunc) {
	if err := n.tree.(*internalTree).root.Walk("", fn, nil); err != nil {
		n.w.Fatal(err)
	}
}
