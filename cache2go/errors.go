package cache2go

import "errors"

var (
	// ErrKeyNotFound gets returned when a specific key couldn't be found
	ErrKeyNotFount = errors.New("Key not found in cache")
	// ErrKeyNotFoundOrLoadable gets retouned when a specific key couldn't be
	// found and loading via the data-loader callback also failed
	ErrKeyNotFoundOrLoadable = errors.New("key not found and could not be loaded into cache")
)