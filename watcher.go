// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Edited by in 2025 olandr.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import "errors"

var (
	errAlreadyWatched  = errors.New("path is already watched")
	errNotWatched      = errors.New("path is not being watched")
	errInvalidEventSet = errors.New("invalid event set provided")
)

// Watcher is a intermediate interface for wrapping inotify, ReadDirChangesW,
// FSEvents, kqueue and poller implementations.
// The Watcher will leverage a "recursive" traversal strategy for those OS, which do support
// recursive watching over directories.
//
// The watcher implementation is expected to do its own mapping between paths and
// create watchers if underlying event notification does not support it. For
// the ease of implementation it is guaranteed that paths provided via Watch and
// Unwatch methods are absolute and clean.
type watcher interface {
	// Watch requests a watcher creation for the given path and given event set.
	Watch(path string, event Event, is_recursive bool) error

	// Unwatch requests a watcher deletion for the given path and given event set.
	Unwatch(path string, is_recursive bool) error

	// Rewatch provides a functionality for modifying existing watch-points, like
	// expanding its event set.
	//
	// Rewatch modifies existing watch-point under for the given path. It passes
	// the existing event set currently registered for the given path, and the
	// new, requested event set.
	//
	// It is guaranteed that tree will not pass to Rewatch zero value for any
	// of its arguments. If old == new and watcher can be upgraded to
	// recursiveWatcher interface, a watch for the corresponding path is expected
	// to be changed from recursive to the non-recursive one.
	Rewatch(oldpath, newpath string, old, new Event, is_recursive bool) error

	// Close unwatches all paths that are registered. When Close returns, it
	// is expected it will report no more events.
	Close() error
}
