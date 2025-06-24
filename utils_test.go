// File created by olandr (c) 2025.
// Contains code from Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

var wd string

func generateDirs(n int) []string {
	var sb strings.Builder
	var ret []string
	for range n {
		for level := n; level > 0; level-- {
			sb.WriteString(fmt.Sprintf("/"))
			sb.WriteString(fmt.Sprintf("%s", gofakeit.LetterN(6)))
		}
		ret = append(ret, generateFakeFileAtDestination(true, sb.String()))
	}
	return ret
}

func generateFakeFileAtRandomDest(is_dir bool, level int) string {
	var sb strings.Builder
	for range level {
		sb.WriteString(fmt.Sprintf("/"))
		sb.WriteString(fmt.Sprintf("%s", gofakeit.LetterN(6)))
	}
	return generateFakeFileAtDestination(is_dir, sb.String())
}

func generateFakeFileAtDestination(is_dir bool, destination string) string {
	name := gofakeit.LetterN(6)
	if !is_dir {
		name = fmt.Sprintf("%v.%v", name, gofakeit.FileExtension())
	}
	return fmt.Sprintf("%v/%v", destination, name)
}

func init() {
	var err error
	if wd, err = os.Getwd(); err != nil {
		panic("Getwd()=" + err.Error())
	}
}

func tmpcreateall(tmp string, path string) error {
	isdir := isDir(path)
	path = filepath.Join(tmp, filepath.FromSlash(path))
	if isdir {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			return err
		}
	}
	return nil
}

func tmpcreate(root, path string) (bool, error) {
	isdir := isDir(path)
	path = filepath.Join(root, filepath.FromSlash(path))
	if isdir {
		if err := os.Mkdir(path, 0755); err != nil {
			return false, err
		}
	} else {
		f, err := os.Create(path)
		if err != nil {
			return false, err
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			return false, err
		}
	}
	return isdir, nil
}

func randomtree(root string) (string, error) {
	var err error
	if root == "" {
		if root, err = os.MkdirTemp(testdata_destination()); err != nil {
			return "", err
		}
	}

	return root, nil
}

func tmptree(root, list string) (string, error) {
	f, err := os.Open(list)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if root == "" {
		if root, err = os.MkdirTemp(testdata_destination()); err != nil {
			return "", err
		}
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := tmpcreateall(root, scanner.Text()); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return root, nil
}

func callern(n int) string {
	_, file, line, ok := runtime.Caller(n)
	if !ok {
		return "<unknown>"
	}
	return filepath.Base(file) + ":" + strconv.Itoa(line)
}

func caller() string {
	return callern(3)
}

func timeout() time.Duration {
	if s := os.Getenv("NOTIFY_TIMEOUT"); s != "" {
		if t, err := time.ParseDuration(s); err == nil {
			return t
		}
	}
	return 2 * time.Second
}

func testdata_destination() (string, string) {
	if s := os.Getenv("NOTIFY_TMP"); s != "" {
		return filepath.Split(s)
	}
	return "testdata", ""
}

func isDir(path string) bool {
	r := path[len(path)-1]
	return r == '\\' || r == '/'
}

func EqualEventInfo(want, got EventInfo) error {
	if got.Event() != want.Event() {
		return fmt.Errorf("want Event()=%v; got %v (path=%s)", want.Event(),
			got.Event(), want.Path())
	}
	path := strings.TrimRight(filepath.FromSlash(want.Path()), `/\`)
	if !strings.HasSuffix(got.Path(), path) {
		return fmt.Errorf("want Path()=%s; got %s (event=%v)", path, got.Path(),
			want.Event())
	}
	return nil
}

func HasEventInfo(want, got Event, p string) error {
	if got&want != want {
		return fmt.Errorf("want Event=%v; got %v (path=%s)", want,
			got, p)
	}
	return nil
}

func EqualCall(want, got Call) error {
	if want.F != got.F {
		return fmt.Errorf("want F=%v; got %v (want.P=%q, got.P=%q)", want.F, got.F, want.P, got.P)
	}
	if got.E != want.E {
		return fmt.Errorf("want E=%v; got %v (want.P=%q, got.P=%q)", want.E, got.E, want.P, got.P)
	}
	if got.NE != want.NE {
		return fmt.Errorf("want NE=%v; got %v (want.P=%q, got.P=%q)", want.NE, got.NE, want.P, got.P)
	}
	if want.C != got.C {
		return fmt.Errorf("want C=%p; got %p (want.P=%q, got.P=%q)", want.C, got.C, want.P, got.P)
	}
	if want := filepath.FromSlash(want.P); !strings.HasSuffix(got.P, want) {
		return fmt.Errorf("want P=%s; got %s", want, got.P)
	}
	if want := filepath.FromSlash(want.NP); !strings.HasSuffix(got.NP, want) {
		return fmt.Errorf("want NP=%s; got %s", want, got.NP)
	}
	return nil
}
func drainall(c chan EventInfo) (ei []EventInfo) {
	time.Sleep(50 * time.Millisecond)
	for {
		select {
		case e := <-c:
			ei = append(ei, e)
			runtime.Gosched()
		default:
			return
		}
	}
}
