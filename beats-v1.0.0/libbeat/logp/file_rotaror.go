package logp

import "os"

type FileRotator struct {
	Path             string
	Name             string
	RotateEveryBytes *uint64
	KeepFiles        *int

	current     *os.File
	currentSize uint64
}
