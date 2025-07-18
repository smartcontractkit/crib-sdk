package port

import (
	"io/fs"
	"iter"
	"os"
)

// FileHandlerFn is a user function that is called for each file found in a directory.
// If the method returns an error, the file is skipped.
type FileHandlerFn func(f FileReader, path string) error

// FileHandler defines adapter methods for interacting with a file system.
type FileHandler interface {
	FileWriter

	// Open opens a file at the given path relative to the FileHandler's root directory.
	Open(name string) (fs.File, error)
}

// FileReader defines read-only methods for interacting with a file system.
type FileReader interface {
	fs.ReadFileFS

	// Name returns the root directory of the file system.
	Name() string
	// AbsPathFor returns an absolute path for the given relative path components.
	// This is the equivalent of `filepath.Join(h.Name(), parts...)` but ensures the path is absolute.
	AbsPathFor(parts ...string) string
	// FileExists checks if a file exists at the given path relative to the FileReader's root directory.
	FileExists(name string) bool
	// DirExists checks if a directory exists at the given path relative to the FileReader's root directory.
	DirExists(name string) bool
	// Scan handles walking a directory, returning an iterator for files that pass
	// the provided handler function.
	Scan(fileFn FileHandlerFn) iter.Seq[string]
}

// FileWriter defines write methods for interacting with a file system.
type FileWriter interface {
	FileReader

	// MkdirAll creates a directory structure with the specified name and permissions.
	MkdirAll(name string, perm fs.FileMode) error
	// Mkdir creates a directory with the specified name and permissions.
	Mkdir(name string, perm fs.FileMode) error
	// Create creates or truncates the named file in the root.
	Create(name string) (*os.File, error)
	// WriteFile writes data to a file at the given path relative to the FileWriter's root directory.
	WriteFile(name string, data []byte) error
}
