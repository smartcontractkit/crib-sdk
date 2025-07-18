package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/common/dry"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
	"github.com/smartcontractkit/crib-sdk/internal/core/port"
)

type (
	// EnvLoader is a [port.ValuesLoader] implementation that loads values from environment variables with the given
	// environment variable prefix.
	EnvLoader struct {
		prefix string
	}

	// YAMLLoader is a [port.ValuesLoader] implementation that loads values from a [bytes.Buffer] containing YAML data.
	YAMLLoader struct {
		parsed map[string]any
	}

	// FileLoader is a [port.ValuesLoader] implementation that attempts to read a file and parse the file with the
	// provided [port.ValuesParser] to return a map of values.
	FileLoader struct {
		file     port.FileHandler
		valuesFn port.ValuesParser
		filePath string
	}
)

// NewHelmValuesLoader is a helper function to create a loader capable of loading values from a Helm Chart values file.
// This function is the equivalent of:
//
//	NewFileLoader(ctx, "path/to/helm/chart/values.yaml", NewYAMLLoader())
//
// Use:
//
//	l, err := NewHelmValuesLoader(ctx, "path/to/helm/chart") // including values.yaml is optional.
//	values, err := l.Values()
func NewHelmValuesLoader(ctx context.Context, path string) (*FileLoader, error) {
	if !strings.HasSuffix(path, domain.HelmValuesFileName) {
		path = filepath.Join(path, domain.HelmValuesFileName)
	}
	return NewFileLoader(ctx, path, NewYAMLLoader())
}

// NewTestYAMLLoader initializes a new YAMLLoader for testing purposes. It uses a map to store parsed values.
// It returns a [port.ValuesLoader] that can be used to retrieve the values directly without parsing from a file.
func NewTestYAMLLoader(values map[string]any) port.ValuesLoader {
	return &YAMLLoader{
		parsed: values,
	}
}

// NewEnvLoader initializes a new EnvLoader with the given environment variable prefix. It handles loading values
// from environment variables that start with the specified prefix. The prefix is stripped from the variable names.
func NewEnvLoader(prefix string) *EnvLoader {
	return &EnvLoader{
		prefix: prefix,
	}
}

// Values returns the values loaded from environment variables that start with the specified prefix. The
// prefix is stripped from the variable names, and the values are returned as a map[string]any.
// Values reads environment variables with the loader's prefix and returns them as a map.
func (e *EnvLoader) Values() (map[string]any, error) {
	result := make(map[string]any)

	// Get all environment variables
	envVars := os.Environ()

	// Filter and process variables with our prefix
	for _, envVar := range envVars {
		key, value, found := strings.Cut(envVar, "=")
		if !found {
			continue
		}

		// Check if the key starts with our prefix
		if !strings.HasPrefix(key, e.prefix+"_") {
			continue
		}

		// Remove the prefix and underscore to get the actual key name
		actualKey := strings.TrimPrefix(key, e.prefix+"_")

		// Parse the value into the appropriate type
		parsed, err := e.parseValue(value)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", key, err)
		}
		result[actualKey] = parsed
	}

	return result, nil
}

func (e *EnvLoader) parseValue(value string) (any, error) {
	// Parse simple types.
	if i, err := strconv.Atoi(value); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d.String(), nil
	}
	if b, err := strconv.ParseBool(value); err == nil {
		return b, nil
	}
	// Parse a map.
	if strings.Contains(value, ":") {
		m := make(map[string]any)
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			k, v, found := strings.Cut(pair, ":")
			if !found {
				continue
			}
			parsedV, err := e.parseValue(v)
			if err != nil {
				return nil, err
			}
			m[k] = parsedV
		}
		return m, nil
	}
	// Parse a slice.
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		res := make([]any, len(parts))
		for i, part := range parts {
			parsedPart, err := e.parseValue(part)
			if err != nil {
				return nil, err
			}
			res[i] = parsedPart
		}
		return res, nil
	}
	return value, nil
}

// NewYAMLLoader initializes a new YAMLLoader. The YAMLLoader implements both the [port.ValuesLoader] and
// [port.ValuesParser] interfaces to allow loading values from a YAML file or a [bytes.Buffer] containing YAML data.
func NewYAMLLoader() *YAMLLoader {
	return &YAMLLoader{
		parsed: make(map[string]any),
	}
}

func (y *YAMLLoader) Parse(r io.Reader) error {
	if r == nil {
		return errors.New("reader cannot be nil")
	}
	if len(y.parsed) > 0 {
		return errors.New("YAMLLoader already has parsed values, cannot parse again")
	}

	// Decode the YAML data from the reader into a map.
	err := yaml.NewDecoder(r).Decode(&y.parsed)
	return dry.Wrapf(err, "decoding YAML data")
}

// Values returns the parsed values from the YAMLLoader. It implements the [port.ValuesLoader.Values] method.
func (y *YAMLLoader) Values() (map[string]any, error) {
	if len(y.parsed) == 0 {
		return nil, errors.New("no values parsed, call Parse() first")
	}
	return y.parsed, nil
}

// NewFileLoaderFromFS initializes a new FileLoader with the given [port.FileHandler].
func NewFileLoaderFromFS(fh port.FileHandler, path string, valuesFn port.ValuesParser) (*FileLoader, error) {
	var err error
	if fh == nil {
		err = errors.Join(err, errors.New("file handler cannot be nil"))
	}
	if valuesFn == nil {
		err = errors.Join(err, errors.New("valuesFn cannot be nil"))
	}
	root := dry.When(fh == nil, "<unknown>", fh.Name())
	return dry.Wrapf2(&FileLoader{
		file:     fh,
		valuesFn: valuesFn,
		filePath: filepath.Base(path),
	}, err, "creating FileLoader with path: %s", root)
}

// NewFileLoader initializes a new FileLoader with the given path to parse and eventually return a key/value mapping
// of the type map[string]any. It accepts a [port.ValuesParser] function to instruct the loader on
// how to parse the values from the file. Examples could be a YAML, TOML, Properties, JSON parser, etc.
func NewFileLoader(ctx context.Context, path string, valuesFn port.ValuesParser) (*FileLoader, error) {
	dir := filepath.Dir(path)
	fh, err := filehandler.New(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("creating file handler for %s: %w", dir, err)
	}
	return NewFileLoaderFromFS(fh, path, valuesFn)
}

// Values implements the [port.ValuesLoader.Values] method. It returns
// the values from the values.yaml file at the given path or an error.
func (f FileLoader) Values() (values map[string]any, err error) {
	if !f.file.FileExists(f.filePath) {
		return nil, fmt.Errorf("file does not exist: %s", f.filePath)
	}
	if !fs.ValidPath(f.filePath) {
		return nil, fmt.Errorf("invalid file path: %s", f.filePath)
	}

	fh, err := f.file.Open(f.filePath)
	if err, ok := dry.ErrorAs[*fs.PathError](err); ok {
		return nil, fmt.Errorf("attempting %s with file %q: %w", err.Op, err.Path, err.Err)
	}
	// Catch other types of errors that may occur when opening the file.
	if err != nil {
		return values, err
	}
	defer func() {
		err = errors.Join(err, fh.Close())
	}()

	// Invoke the provided valuesFn to parse the file.
	if err := f.valuesFn.Parse(fh); err != nil {
		return nil, dry.Wrapf(err, "decoding values from file: %s", f.filePath)
	}
	vals, err := f.valuesFn.Values()
	return dry.Wrapf2(vals, err, "getting values from file: %s", f.filePath)
}

// SetValueAtPath sets a value at the specified dot path in the map.
// For example, "containers[0].env[0].value" will set the value at that path.
// nolint
func SetValueAtPath(m map[string]any, path string, value any) map[string]any {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return m
	}

	current := m
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if idx := strings.Index(part, "["); idx != -1 {
			// Handle array index
			baseKey := part[:idx]
			idxStr := part[idx+1 : len(part)-1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return m
			}

			// Get or create array
			arr, ok := current[baseKey].([]any)
			if !ok {
				arr = make([]any, idx+1)
				current[baseKey] = arr
			} else if len(arr) <= idx {
				// Extend array if needed
				newArr := make([]any, idx+1)
				copy(newArr, arr)
				arr = newArr
				current[baseKey] = arr
			}

			// Get or create map at index
			if arr[idx] == nil {
				arr[idx] = make(map[string]any)
			}
			next, ok := arr[idx].(map[string]any)
			if !ok {
				next = make(map[string]any)
				arr[idx] = next
			}
			current = next
			continue
		}

		// Handle regular map key
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}

	// Set the final value
	lastPart := parts[len(parts)-1]
	if idx := strings.Index(lastPart, "["); idx != -1 {
		// Handle array index for final value
		baseKey := lastPart[:idx]
		idxStr := lastPart[idx+1 : len(lastPart)-1]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return m
		}

		arr, ok := current[baseKey].([]any)
		if !ok {
			arr = make([]any, idx+1)
			current[baseKey] = arr
		} else if len(arr) <= idx {
			newArr := make([]any, idx+1)
			copy(newArr, arr)
			arr = newArr
			current[baseKey] = arr
		}
		arr[idx] = value
	} else {
		current[lastPart] = value
	}

	return m
}
