package filehandler

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

func TestISatisfiesFileHandler(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	is.Implements((*port.FileHandler)(nil), &Handler{}, "Handler should implement port.FileHandler interface")
	is.Implements((*port.FileReader)(nil), &Handler{}, "Handler should implement port.FileReader interface")
	is.Implements((*port.FileWriter)(nil), &Handler{}, "Handler should implement port.FileWriter interface")
}

func TestCopyDir(t *testing.T) {
	t.Parallel()

	src := fstest.MapFS{
		"test.txt":  &fstest.MapFile{},
		"test2.txt": &fstest.MapFile{},
	}

	srch, err := NewReadOnlyFS(src)
	require.NoError(t, err)
	dsth, err := New(t.Context(), t.TempDir())
	require.NoError(t, err)

	assert.NoError(t, CopyDir(srch, dsth))

	_, err = src.Open("test.txt")
	assert.NoError(t, err)
	_, err = src.Open("test2.txt")
	assert.NoError(t, err)
}

func TestCopyFile(t *testing.T) {
	t.Parallel()

	src := fstest.MapFS{
		"test.txt": &fstest.MapFile{
			Data: []byte("test data"),
			Mode: 0o644,
		},
	}

	srch, err := NewReadOnlyFS(src)
	require.NoError(t, err)
	dsth, err := New(t.Context(), t.TempDir())
	require.NoError(t, err)

	assert.NoError(t, CopyFile(srch, dsth, "test.txt"))

	raw, err := dsth.ReadFile("test.txt")
	assert.NoError(t, err)
	assert.Equal(t, src["test.txt"].Data, raw)
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		root    string
		err     assert.ErrorAssertionFunc
		handler assert.ValueAssertionFunc
		eq      assert.ComparisonAssertionFunc
		dir     assert.BoolAssertionFunc
	}{
		{
			name:    "success",
			root:    t.TempDir(),
			err:     assert.NoError,
			handler: assert.NotNil,
			eq:      assert.Equal,
			dir:     assert.True,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, err := New(t.Context(), tc.root)
			tc.err(t, err)
			if err != nil {
				var pathErr *os.PathError
				assert.ErrorAs(t, err, &pathErr)
			}
			if !tc.handler(t, h) {
				tc.eq(t, h.Name(), tc.root)
				tc.dir(t, assert.DirExists(t, h.Name()))
				f, err := h.Create(gofakeit.Noun())
				t.Cleanup(func() {
					assert.NoError(t, f.Close())
				})
				assert.NoError(t, err)
				assert.NotNil(t, f)
			}
		})
	}
}

func TestNewTempHandler(t *testing.T) {
	t.Parallel()
	is := assert.New(t)

	tests := []struct {
		name       string
		prefix     string
		handlerErr assert.ErrorAssertionFunc
		handler    assert.ValueAssertionFunc
		dir        assert.BoolAssertionFunc
	}{
		{
			name:       "no prefix",
			handlerErr: assert.NoError,
			handler:    assert.NotNil,
			dir:        assert.True,
		},
		{
			name:       "one word",
			prefix:     gofakeit.Noun(),
			handlerErr: assert.NoError,
			handler:    assert.NotNil,
			dir:        assert.True,
		},
		{
			name:       "two words",
			prefix:     strings.Join([]string{gofakeit.Adjective(), gofakeit.Noun()}, " "),
			handlerErr: assert.NoError,
			handler:    assert.NotNil,
			dir:        assert.True,
		},
		{
			name:       "split by /",
			prefix:     filepath.Join(gofakeit.Noun(), gofakeit.Adjective()),
			handlerErr: assert.NoError,
			handler:    assert.NotNil,
			dir:        assert.True,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, err := NewTempHandler(t.Context(), tc.prefix)
			tc.handlerErr(t, err)
			tc.handler(t, h)
			if err != nil {
				return
			}

			tc.dir(t, is.DirExists(h.Name()))
			prefix := normalizePrefix(tc.prefix)
			is.Truef(strings.HasPrefix(filepath.Base(h.Name()), prefix), "Prefix mismatch: got %s, want %s", filepath.Base(h.Name()), prefix)

			f, err := h.Create(gofakeit.Noun())
			t.Cleanup(func() {
				is.NoError(f.Close())
			})
			is.NoError(err)
			is.NotNil(f)
		})
	}
}

func TestNewReadOnlyFS(t *testing.T) {
	t.Parallel()
	is := assert.New(t)
	must := require.New(t)

	fs := fstest.MapFS{
		"test.txt": &fstest.MapFile{
			Data: []byte("test data"),
		},
	}

	ro, err := NewReadOnlyFS(fs)
	must.NoError(err)
	must.NotNil(ro)

	t.Run("read", func(t *testing.T) {
		t.Parallel()

		data, err := ro.ReadFile("test.txt")
		is.NoError(err)
		is.Equal([]byte("test data"), data)
	})
	t.Run("attempt write", func(t *testing.T) {
		t.Parallel()

		_, err := ro.Create("newfile.txt")
		is.ErrorIs(err, domain.ErrReadOnlyFileSystem)
	})
}

func TestScan(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"test.txt":       &fstest.MapFile{},
		"test2.txt":      &fstest.MapFile{},
		"test.yaml":      &fstest.MapFile{},
		"test.yml":       &fstest.MapFile{},
		"test2.yaml":     &fstest.MapFile{},
		"a/b/c.txt":      &fstest.MapFile{},
		"a/b/c.yaml":     &fstest.MapFile{},
		"a/b/c/d.txt":    &fstest.MapFile{},
		"a/b/c/test.txt": &fstest.MapFile{},
	}

	tests := []struct {
		desc   string
		fileFn port.FileHandlerFn
		want   []string
	}{
		{
			desc: "Want all files",
			fileFn: func(_ port.FileReader, path string) error {
				return nil
			},
			want: []string{
				"test.txt",
				"test2.txt",
				"test.yaml",
				"test.yml",
				"test2.yaml",
				"a/b/c.txt",
				"a/b/c.yaml",
				"a/b/c/d.txt",
				"a/b/c/test.txt",
			},
		},
		{
			desc: "Want only yaml files",
			fileFn: func(_ port.FileReader, path string) error {
				ext := filepath.Ext(path)
				if ext == ".yaml" || ext == ".yml" {
					return nil
				}
				return domain.NewSkipFileError(path)
			},
			want: []string{
				"test.yaml",
				"test.yml",
				"test2.yaml",
				"a/b/c.yaml",
			},
		},
		{
			desc: "Want only txt files",
			fileFn: func(_ port.FileReader, path string) error {
				ext := filepath.Ext(path)
				if ext == ".txt" {
					return nil
				}
				return domain.NewSkipFileError(path)
			},
			want: []string{
				"test.txt",
				"test2.txt",
				"a/b/c.txt",
				"a/b/c/d.txt",
				"a/b/c/test.txt",
			},
		},
		{
			desc: "Want only files in a specific directory",
			fileFn: func(_ port.FileReader, path string) error {
				if filepath.Dir(path) == "a/b" {
					return nil
				}
				return domain.NewSkipFileError(path)
			},
			want: []string{
				"a/b/c.txt",
				"a/b/c.yaml",
			},
		},
		{
			desc: "Want only files with specific name",
			fileFn: func(_ port.FileReader, path string) error {
				if filepath.Base(path) == "test.txt" {
					return nil
				}
				return domain.NewSkipFileError(path)
			},
			want: []string{
				"test.txt",
				"a/b/c/test.txt",
			},
		},
		{
			desc: "No files found",
			fileFn: func(_ port.FileReader, path string) error {
				return domain.NewSkipFileError(path)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			h := &Handler{
				rootFS:     fs,
				fileReader: NewBufferedReader(fs, DefaultReader),
			}
			got := slices.Collect(h.Scan(tc.fileFn))
			slices.Sort(got)
			slices.Sort(tc.want)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"test.txt": &fstest.MapFile{},
	}

	h := &Handler{
		rootFS:     fs,
		fileReader: NewBufferedReader(fs, DefaultReader),
	}

	tests := []struct {
		desc string
		file string
		err  assert.ErrorAssertionFunc
		eq   assert.ValueAssertionFunc
	}{
		{
			desc: "Open existing file",
			file: "test.txt",
			err:  assert.NoError,
			eq:   assert.NotNil,
		},
		{
			desc: "Open non-existing file",
			file: "nonexistent.txt",
			err:  assert.Error,
			eq:   assert.Nil,
		},
		{
			desc: "Directory traversal attempt",
			file: "../../test.txt",
			err:  assert.Error,
			eq:   assert.Nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			f, err := h.Open(tc.file)
			t.Cleanup(func() {
				if f != nil {
					assert.NoError(t, f.Close())
				}
			})
			tc.err(t, err)
			tc.eq(t, f)
		})
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"test.txt": &fstest.MapFile{
			Data: []byte("test data"),
			Mode: 0o644,
		},
		"empty.txt": &fstest.MapFile{},
	}

	h := &Handler{
		rootFS:     fs,
		fileReader: NewBufferedReader(fs, DefaultReader),
	}

	tests := []struct {
		desc string
		file string
		err  assert.ErrorAssertionFunc
		eq   assert.ValueAssertionFunc
		val  assert.ComparisonAssertionFunc
	}{
		{
			desc: "Read existing file",
			file: "test.txt",
			err:  assert.NoError,
			eq:   assert.NotNil,
			val:  assert.Equal,
		},
		{
			desc: "Read empty file",
			file: "empty.txt",
			err:  assert.NoError,
			eq:   assert.NotNil,
			val:  assert.NotEqual,
		},
		{
			desc: "Read non-existing file",
			file: "nonexistent.txt",
			err:  assert.Error,
			eq:   assert.Nil,
			val:  assert.NotEqual,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			data, err := h.ReadFile(tc.file)
			tc.err(t, err)
			tc.eq(t, data)
			tc.val(t, string(data), "test data")
		})
	}
}

func TestWriteFile(t *testing.T) {
	t.Parallel()
	must := require.New(t)
	is := assert.New(t)

	content := gofakeit.LoremIpsumSentence(10)

	h, err := NewTempHandler(t.Context(), "test-write")
	must.NoError(err, "Creating temporary file handler should not return an error")

	for _, file := range []string{"test.txt", "a/test.txt"} {
		t.Run(file, func(t *testing.T) {
			t.Run("Write file", func(t *testing.T) {
				must.NoError(h.WriteFile(file, []byte(content)))
			})
			t.Run("File exists", func(t *testing.T) {
				dir := filepath.Dir(file)
				must.Truef(h.FileExists(file), "%s should exist after writing", file)
				must.Truef(h.DirExists(dir), "Directory %q should exist after writing", dir)
			})
			t.Run("Read after write matches content", func(t *testing.T) {
				data, err := h.ReadFile(file)
				must.NoError(err, "Reading file %s should not return an error", file)
				is.Equal(content, string(data), "Content of %s should match written content", file)
			})
		})
	}
}

func TestAbsPathFor(t *testing.T) {
	t.Parallel()

	fs := fstest.MapFS{
		"test.txt":         &fstest.MapFile{},
		"subdir/test2.txt": &fstest.MapFile{},
	}

	h := &Handler{
		rootFS:     fs,
		fileReader: NewBufferedReader(fs, DefaultReader),
	}

	tests := []struct {
		desc string
		path string
		eq   assert.ComparisonAssertionFunc
	}{
		{
			desc: "Absolute path for existing file",
			path: "test.txt",
			eq:   assert.Equal,
		},
		{
			desc: "Absolute path for file in subdirectory",
			path: "subdir/test2.txt",
			eq:   assert.Equal,
		},
		{
			desc: "Attempted directory traversal",
			path: "/../../test.txt",
			eq:   assert.NotEqual,
		},
		{
			desc: "no arguments",
			eq:   assert.Equal,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			var absPath string
			if tc.path == "" {
				absPath = h.AbsPathFor()
			} else {
				absPath = h.AbsPathFor(tc.path)
			}

			tc.eq(t, absPath, filepath.Join(h.Name(), tc.path))
			t.Logf("Absolute path for %s: %s", tc.path, absPath)
		})
	}
}

func Test_normalizePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"-", ""},
		{"--foo", "foo"},
		{"- bar", "bar"},
		{"hello", "hello"},
		{"hello world", "hello_world"},
		{"foo/bar", "foo_bar"},
		{"/leading", "leading"},
		{"\u200Bfoo", "foo"},   // zero-width space
		{"emojis👍🏼", "emojis"}, // emojis and modifiers
		{"\u0301e", "e"},       // combining acute
		{"@<>|:*?\"\\/", ""},   // all forbidden characters
		{strings.Repeat("-", 100), ""},
		{
			// long string with replacements
			strings.Repeat("s/", 100),
			strings.Repeat("s_", 32)[:63], // Last character is removed.
		},
		{
			// long string
			strings.Repeat("Na Na ", 12) + "Batman",
			strings.Repeat("Na_Na_", 12)[:64],
		},
		// Attempt directory traversal.
		{"/../foo", "foo"},
		{"foo/../bar", "bar"}, // See https://9p.io/sys/doc/lexnames.html for why.
		{"../foo", "foo"},
		{"./foo", "foo"},
		{"foo/./bar", "foo_bar"},
		{"./../foo", "foo"},
		{"foo/../../bar", "bar"}, // See https://9p.io/sys/doc/lexnames.html for why.
		{"/tmp/../foo", "foo"},   // See https://9p.io/sys/doc/lexnames.html for why.
		{"foo//bar", "foo_bar"},
		// testdata/fuzz/Fuzz_normalizePrefix/5a27b77bcac9e86f
		{"0 -", "0"},
		// testdata/fuzz/Fuzz_normalizePrefix/a016ef77d201ccb4
		{" - 0", "0"},
		// FuzzNewTempHandler/f42f19ae9dc9a943
		{"00000\U000113fc", "00000"}, // U+113FC is a valid character
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			result := normalizePrefix(tc.input)
			assert.True(t, len(result) <= 64, "result should not exceed 64 characters")
			assert.Equal(t, tc.expected, result)
			assert.NotContains(t, result, "-", "result should not contain hyphen at the start")
			assert.NotContains(t, result, " /\\:*?\"<>|", "result should not contain forbidden characters")
			assert.True(t, utf8.ValidString(result), "result should be valid UTF-8")
		})
	}
}

func Fuzz_normalizePrefix(f *testing.F) {
	seeds := []string{
		"",
		"-",
		"--foo",
		"- bar",
		"hello",
		"foo/bar",
		"/leading",
		"\u200Bfoo",    // zero-width space
		"emojis👍🏼",     // emojis and modifiers
		"\u0301e",      // combining acute
		"@<>|:*?\"\\/", // all forbidden
		strings.Repeat("-", 100),
		strings.Repeat("s/", 100), // long string
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	const forbidden = " /\\:*?\"<>|@"
	f.Fuzz(func(t *testing.T, prefix string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on input: %q, panic: %v", prefix, r)
			}
		}()
		result := normalizePrefix(prefix)

		// must not start with a hyphen
		if strings.HasPrefix(result, "-") {
			t.Errorf("result starts with hyphen: input=%q, output=%q", prefix, result)
		}
		// must not start or end with whitespace
		if strings.HasPrefix(result, " ") || strings.HasSuffix(result, " ") {
			t.Errorf("result starts or ends with whitespace: input=%q, output=%q", prefix, result)
		}
		// must not start or end with an underscore
		if strings.HasPrefix(result, "_") || strings.HasSuffix(result, "_") {
			t.Errorf("result starts or ends with underscore: input=%q, output=%q", prefix, result)
		}
		// must not be greater than 64 characters
		if len(result) > 64 {
			t.Errorf("result is longer than 64 characters: input=%q, output=%q", prefix, result)
		}
		// must not contain forbidden characters
		if i := strings.IndexAny(result, forbidden); i != -1 {
			t.Errorf("forbidden char in result: input=%q, output=%q, char=%q", prefix, result, result[i])
		}
		// output must be valid utf8
		if !utf8.ValidString(result) {
			t.Errorf("output is invalid utf8: input=%q, output=%q", prefix, result)
		}
	})
}

func FuzzNewTempHandler(f *testing.F) {
	seeds := []string{
		"",
		"-",
		"--foo",
		"- bar",
		"hello",
		"foo/bar",
		"/leading",
		"\u200Bfoo",    // zero-width space
		"emojis👍🏼",     // emojis and modifiers
		"\u0301e",      // combining acute
		"@<>|:*?\"\\/", // all forbidden
		strings.Repeat("-", 100),
		strings.Repeat("s/", 100), // long string
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, prefix string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on input: %q, panic: %v", prefix, r)
			}
		}()

		fh, err := NewTempHandler(t.Context(), prefix)
		if err != nil {
			t.Fatalf("error creating temp handler input=%q: %v", prefix, err)
		}
		t.Cleanup(func() {
			if err := fh.RemoveAll(); err != nil {
				t.Errorf("error removing temp handler input=%q: %v", prefix, err)
			}
		})

		normalized := normalizePrefix(prefix)
		got := filepath.Base(fh.Name())

		parts := strings.Split(got, "-")
		if len(parts) < 2 {
			t.Fatalf("expected handler name to contain a hyphen input=%q, output=%q: got %q", prefix, got, got)
		}

		suffix := parts[len(parts)-1]
		newPrefix := strings.TrimPrefix(strings.Join(parts[:len(parts)-1], "-"), "crib-sdk")

		t.Logf("prefix: %s, random: %s", newPrefix, suffix)
		if suffix == "" {
			t.Errorf("expected handler name to contain a random suffix input=%q, output=%q: got %q", prefix, got, got)
		}
		if !strings.EqualFold(newPrefix, normalized) {
			t.Errorf("unexpected handler name input=%q, output=%q: got %q, want prefix %q", prefix, got, newPrefix, normalized)
		}
	})
}
