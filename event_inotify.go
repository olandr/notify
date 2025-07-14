// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build linux
// +build linux

package notify

import "golang.org/x/sys/unix"

// Platform independent event values.
const (
	osSpecificCreate Event = 0x100000 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename
	// internal
	// recursive is used to distinguish recursive eventsets from non-recursive ones
	recursive
	// omit is used for dispatching internal events; only those events are sent
	// for which both the event and the watchpoint has omit in theirs event sets.
	omit
)

// Inotify specific masks are legal, implemented events that are guaranteed to
// work with notify package on linux-based systems.
const (
	InAccess       = Event(unix.IN_ACCESS)        // File was accessed
	InModify       = Event(unix.IN_MODIFY)        // File was modified
	InAttrib       = Event(unix.IN_ATTRIB)        // Metadata changed
	InCloseWrite   = Event(unix.IN_CLOSE_WRITE)   // Writtable file was closed
	InCloseNowrite = Event(unix.IN_CLOSE_NOWRITE) // Unwrittable file closed
	InOpen         = Event(unix.IN_OPEN)          // File was opened
	InMovedFrom    = Event(unix.IN_MOVED_FROM)    // File was moved from X
	InMovedTo      = Event(unix.IN_MOVED_TO)      // File was moved to Y
	InCreate       = Event(unix.IN_CREATE)        // Subfile was created
	InDelete       = Event(unix.IN_DELETE)        // Subfile was deleted
	InDeleteSelf   = Event(unix.IN_DELETE_SELF)   // Self was deleted
	InMoveSelf     = Event(unix.IN_MOVE_SELF)     // Self was moved
)

var osestr = map[Event]string{
	InAccess:       "notify.InAccess",
	InAttrib:       "notify.InAttrib",
	InCloseNowrite: "notify.InCloseNowrite",
	InCloseWrite:   "notify.InCloseWrite",
	InCreate:       "notify.InCreate",
	InDelete:       "notify.InDelete",
	InDeleteSelf:   "notify.InDeleteSelf",
	InModify:       "notify.InModify",
	InMovedFrom:    "notify.InMovedFrom",
	InMovedTo:      "notify.InMovedTo",
	InMoveSelf:     "notify.InMoveSelf",
	InOpen:         "notify.InOpen",
}

var osstre = map[string]Event{
	"notify.InAccess":       InAccess,
	"notify.InAttrib":       InAttrib,
	"notify.InCloseNowrite": InCloseNowrite,
	"notify.InCloseWrite":   InCloseWrite,
	"notify.InCreate":       InCreate,
	"notify.InDelete":       InDelete,
	"notify.InDeleteSelf":   InDeleteSelf,
	"notify.InModify":       InModify,
	"notify.InMovedFrom":    InMovedFrom,
	"notify.InMovedTo":      InMovedTo,
	"notify.InMoveSelf":     InMoveSelf,
	"notify.InOpen":         InOpen,
}

// Inotify behavior events are not **currently** supported by notify package.
const (
	inDontFollow = Event(unix.IN_DONT_FOLLOW)
	inExclUnlink = Event(unix.IN_EXCL_UNLINK)
	inMaskAdd    = Event(unix.IN_MASK_ADD)
	inOneshot    = Event(unix.IN_ONESHOT)
	inOnlydir    = Event(unix.IN_ONLYDIR)
)

type event struct {
	sys       unix.InotifyEvent
	path      string
	event     Event
	timestamp int64
}

func (e *event) Timestamp() int64     { return e.timestamp }
func (e *event) Event() Event         { return e.event }
func (e *event) Path() string         { return e.path }
func (e *event) Sys() interface{}     { return &e.sys }
func (e *event) isDir() (bool, error) { return e.sys.Mask&unix.IN_ISDIR != 0, nil }
