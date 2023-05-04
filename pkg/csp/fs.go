package csp

import (
	"errors"
	"io/fs"
	"math"
	"net"
	"time"
)

type FS struct {
	conn net.Conn
}

func (fs FS) Open(name string) (fs.File, error) {
	if fs.conn == nil {
		return File{}, errors.New("TODO")
	}
	return File{
		conn: fs.conn,
		name: name,
	}, nil
}

type File struct {
	name string
	conn net.Conn
}

func (f File) Stat() (fs.FileInfo, error) {
	return FileInfo{
		name: f.name,
	}, nil
}

func (f File) Read(b []byte) (int, error) {
	return f.conn.Read(b)
}

func (f File) Close() error {
	return f.conn.Close()
}

type FileInfo struct {
	name string
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	return math.MinInt64 // TODO
}

func (fi FileInfo) Mode() fs.FileMode {
	return fs.ModeSocket
}

func (fi FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (fi FileInfo) IsDir() bool {
	return false
}

func (fi FileInfo) Sys() any {
	return nil
}
