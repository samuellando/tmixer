package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shopify/go-lua"
)

func absPath(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("while getting home dir: %w", err)
	}
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("while converting to abs path: %w", err)
	}
	return abs, nil
}

func mergeTable(l *lua.State, to, from int) {
	l.PushNil()
	for l.Next(from - 1) {
		key := lua.CheckString(l, -2)
		l.SetField(to-2, key)
	}
}

func merge(l *lua.State) int {
	l.CreateTable(0, 0)
	mergeTable(l, -1, -3)
	mergeTable(l, -1, -2)
	return 1
}

func loadSubDirectories(l *lua.State) int {
	dir, _ := absPath(lua.CheckString(l, -3))
	prefix := lua.CheckString(l, -2)
	dirEntries, _ := os.ReadDir(dir)
	l.CreateTable(0, 0)
	for _, f := range dirEntries {
		if f.IsDir() {
			l.CreateTable(0, 0)
			l.PushString(filepath.Join(dir, f.Name()))
			l.SetField(-2, "directory")
			mergeTable(l, -1, -3)
			l.SetField(-2, prefix+f.Name())
		}
	}
	return 1
}

func main() {
	l := lua.NewState()
	lua.OpenLibraries(l)
	l.PushGoFunction(loadSubDirectories)
	l.SetGlobal("loadSubDirectories")
	l.PushGoFunction(merge)
	l.SetGlobal("merge")
	st := time.Now()
	if err := lua.DoFile(l, "hello.lua"); err != nil {
		panic(err)
	}
	ListProjects(l)
	fmt.Println(time.Since(st))
	st = time.Now()
	GetProject(l, "project--tmixer")
	fmt.Println(time.Since(st))
}

func ListProjects(l *lua.State) {
	l.Global("tmixer")
	defer l.Pop(1)
	if !l.IsTable(-1) {
		panic("Expected 'tmixer' to be a lua table")
	}
	l.Field(-1, "projects")
	defer l.Pop(1)
	if l.IsNil(-1) {
		return
	}
	if !l.IsTable(-1) {
		panic("Expected 'tmixer.projects' to be a lua table")
	}
	l.PushNil()
	for l.Next(-2) {
		key := lua.CheckString(l, -2)
		fmt.Println(key)
		l.Pop(1)
	}
}

func GetProject(l *lua.State, name string) {
	l.Global("tmixer")
	defer l.Pop(1)
	if !l.IsTable(-1) {
		panic("Expected 'tmixer' to be a lua table")
	}
	l.Field(-1, "projects")
	defer l.Pop(1)
	if l.IsNil(-1) {
		return
	}
	l.Field(-1, name)
	defer l.Pop(1)
	if l.IsNil(-1) {
		return
	}
	l.Field(-1, "switch")
	defer l.Pop(1)
	if l.IsNil(-1) {
		return
	}
	l.Call(0, 0)
}
