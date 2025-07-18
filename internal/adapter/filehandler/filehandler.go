// Package filehandler is an adapter for handling file operations.
package filehandler

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

// Handler is the primary struct for file handling. It implements interfaces defined
// in [port.FileHandler]. The Handler is rooted at a specific directory, defined
// at initialization time.
type Handler struct {
	root   *os.Root
	rootFS fs.FS

	fileReader *BufferedReader
}

// CopyDir copies a directory from src to dst.
func CopyDir(src port.FileReader, dst port.FileWriter) error {
	allFilesFn := func(port.FileReader, string) error {
		return nil
	}
	for file := range src.Scan(allFilesFn) {
		err := dry.FirstErrorFns(
			func() error { return dst.MkdirAll(filepath.Dir(file), os.ModePerm) },
			func() error { return CopyFile(src, dst, file) },
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// CopyFile copies a file from src to dst. It takes a source and destination file system handler and a file name.
func CopyFile(src port.FileReader, dst port.FileWriter, file string) (err error) {
	// Open the source file for reading.
	srcFile, err := src.Open(file)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, srcFile.Close())
	}()

	// Create the destination file for writing.
	dstFile, err := dst.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, dstFile.Close())
	}()

	// Copy the contents from the source file to the destination file.
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}

// New creates a new Handler instance. It takes a context and a root directory.
func New(ctx context.Context, root string) (*Handler, error) {
	dir, err := os.OpenRoot(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if dir == nil {
		// Attempt to create the directory if it does not exist.
		if err := os.MkdirAll(root, 0o750); err != nil {
			return nil, errors.Join(err, os.ErrNotExist)
		}
		return New(ctx, root)
	}

	return &Handler{
		root:       dir,
		rootFS:     dir.FS(),
		fileReader: NewBufferedReader(dir.FS(), DefaultReader),
	}, nil
}

// NewTempHandler creates a new Handler instance with a temporary directory. It accepts a prefix to
// be used when creating the temporary directory.
func NewTempHandler(ctx context.Context, prefix string) (*Handler, error) {
	prefix = normalizePrefix(prefix)
	if prefix == "" {
		prefix = "crib-sdk"
	}
	dir, err := os.MkdirTemp("", prefix+"-*")
	if err != nil {
		return nil, err
	}
	return New(ctx, dir)
}

// NewReadOnlyFS creates a new Handler instance with a read-only file system. It takes an existing fs.FS as input.
// This is useful for creating a Handler that does not allow writing to the file system, such as for tests.
func NewReadOnlyFS(f fs.FS) (*Handler, error) {
	return &Handler{
		rootFS:     f,
		fileReader: NewBufferedReader(f, DefaultReader),
	}, nil
}

// FS returns the underlying fs.FS of the Handler.
func (h *Handler) FS() fs.FS {
	if h.rootFS == nil {
		return os.DirFS(".")
	}
	return h.rootFS
}

// Name returns the root directory of the Handler.
func (h *Handler) Name() string {
	if h.root == nil {
		return "."
	}
	return h.root.Name()
}

// AbsPathFor returns an absolute path for the given relative path components.
// This is the equivalent of `filepath.Join(h.Name(), parts...)` but ensures the path is absolute.
func (h *Handler) AbsPathFor(parts ...string) string {
	// If the root is nil, return the current directory.
	if h.root == nil {
		return path.Clean(filepath.Join(".", filepath.Join(parts...)))
	}
	// Join the parts with the root directory.
	return path.Clean(filepath.Join(h.root.Name(), filepath.Join(parts...)))
}

// FileExists checks if a file exists at the given path relative to the Handler's root directory.
func (h *Handler) FileExists(name string) bool {
	if h.root == nil {
		return false
	}
	_, err := h.root.Open(name)
	return !errors.Is(err, os.ErrNotExist)
}

// DirExists checks if a directory exists at the given path relative to the Handler's root directory.
func (h *Handler) DirExists(name string) bool {
	if h.root == nil {
		return false
	}
	info, err := h.root.Stat(name)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Scan walks the file tree rooted at the Handler's root directory. It calls
// the provided function for each file found, skipping directories and other special
// file types. Typically, the passed function should make decisions on whether to
// yield the file path or not. Be aware, that if you decide to read a file that you
// may doubly read the file at a later time.
func (h *Handler) Scan(fileFn port.FileHandlerFn) iter.Seq[string] {
	return func(yield func(string) bool) {
		_ = fs.WalkDir(h.rootFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if err := fileFn(h, path); err == nil {
				yield(path)
			}
			return nil
		})
	}
}

// Open implements the Open method in the fs.FS interface. It opens a file at the
// given name relative to the Handler's root directory. It returns a fs.File and an
// error if any.
func (h *Handler) Open(name string) (fs.File, error) {
	return h.fileReader.Open(name)
}

// ReadFile implements the ReadFile method in the fs.FS interface. It reads a file
// at the given name relative to the Handler's root directory. It returns the file
// contents as a byte slice and an error if any.
func (h *Handler) ReadFile(name string) ([]byte, error) {
	return h.fileReader.ReadFile(name)
}

// WriteFile writes data to a file at the given name relative to the Handler's root directory.
// If the file does not exist, it will be created. If the file exists, it will be truncated.
func (h *Handler) WriteFile(name string, data []byte) (err error) {
	if h.root == nil {
		return domain.ErrReadOnlyFileSystem
	}
	dir := filepath.Dir(name)
	// Ensure the directory exists before writing the file.
	if err := h.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}
	f, err := h.Create(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("opening file %q: %w", name, err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()
	_, err = f.Write(data)
	return dry.Wrapf(err, "writing file %q", name)
}

// MkdirAll creates a directory structure with the specified name and permissions.
func (h *Handler) MkdirAll(name string, perm fs.FileMode) error {
	if h.root == nil {
		return domain.ErrReadOnlyFileSystem
	}
	return os.MkdirAll(filepath.Join(h.root.Name(), name), perm)
}

// Mkdir creates a directory with the specified name and permissions.
func (h *Handler) Mkdir(name string, perm fs.FileMode) error {
	if h.root == nil {
		return domain.ErrReadOnlyFileSystem
	}
	return h.root.Mkdir(name, perm)
}

// Create creates or truncates the named file in the root.
func (h *Handler) Create(name string) (*os.File, error) {
	if h.root == nil {
		return nil, domain.ErrReadOnlyFileSystem
	}
	return h.root.Create(name)
}

// RemoveAll deletes the entire file tree rooted at the Handler's root directory.
// This can be a highly destructive action if used incorrectly. It's provided primarily
// for the TempHandler, which is used for temporary file operations.
func (h *Handler) RemoveAll() error {
	if h.root == nil {
		return domain.ErrReadOnlyFileSystem
	}
	return os.RemoveAll(h.root.Name())
}

func normalizePrefix(prefix string) string {
	// Clean the prefix.
	prefix = filepath.Clean(prefix)
	// Trim leading and trailing - and _ from the prefix.
	prefix = strings.Trim(prefix, "._-")
	// Remove leading and trailing whitespace.
	prefix = strings.TrimSpace(prefix)
	// Replace any invalid characters with underscores.
	prefix = strings.Map(func(r rune) rune {
		isInvalid := cmp.Or(
			unicode.IsSymbol(r),
			unicode.IsMark(r),
			r == ' ',
			r == '/',
			r == '\\',
			r == ':',
			r == '*',
			r == '?',
			r == '"',
			r == '<',
			r == '>',
			r == '|',
			r == '@',
			r == '\x00',
			r == '\u0001', // Start of Heading
			r == '\u200B', // Zero Width Space
			r == '\u0301', // Combining Acute Accent
			!unicode.IsGraphic(r),
		)
		if isInvalid {
			return '_'
		}
		return r
	}, prefix)
	// Cap the length to 64 characters.
	if len(prefix) > 64 {
		prefix = prefix[:64]
	}
	prefix = strings.ToValidUTF8(prefix, "_")

	// Ensure the prefix does not start or end with a dot, hyphen, or underscore, and does not contain
	// only those symbols.
	return strings.Trim(prefix, "._-")
}

// Ensure FileHandler implements the FileHandler interface.
var _ port.FileHandler = (*Handler)(nil) //nolint:decorder // Adds a compile-time check to ensure that Handler implements the FileHandler interface.
