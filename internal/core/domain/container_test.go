package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImageURI(t *testing.T) {
	t.Parallel()

	t.Run("repository with tag", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("nginx:1.20")
		is.NoError(err)
		is.Equal("nginx", result.Repository)
		is.Equal("1.20", result.Tag)
		is.Equal("", result.Digest)
	})

	t.Run("repository with digest", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("nginx@sha256:abcd1234")
		is.NoError(err)
		is.Equal("nginx", result.Repository)
		is.Equal("", result.Tag)
		is.Equal("sha256:abcd1234", result.Digest)
	})

	t.Run("repository without tag defaults to latest", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("nginx")
		is.NoError(err)
		is.Equal("nginx", result.Repository)
		is.Equal("latest", result.Tag)
		is.Equal("", result.Digest)
	})

	t.Run("registry with port and tag", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("localhost:5000/nginx:1.20")
		is.NoError(err)
		is.Equal("localhost:5000/nginx", result.Repository)
		is.Equal("1.20", result.Tag)
		is.Equal("", result.Digest)
	})

	t.Run("registry with port without tag", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("localhost:5000/nginx")
		is.NoError(err)
		is.Equal("localhost:5000/nginx", result.Repository)
		is.Equal("latest", result.Tag)
		is.Equal("", result.Digest)
	})

	t.Run("registry with port and digest", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("localhost:5000/nginx@sha256:abcd1234")
		is.NoError(err)
		is.Equal("localhost:5000/nginx", result.Repository)
		is.Equal("", result.Tag)
		is.Equal("sha256:abcd1234", result.Digest)
	})

	t.Run("complex repository path with tag", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		result, err := ParseImageURI("gcr.io/project/subfolder/image:v1.0.0")
		is.NoError(err)
		is.Equal("gcr.io/project/subfolder/image", result.Repository)
		is.Equal("v1.0.0", result.Tag)
		is.Equal("", result.Digest)
	})

	t.Run("invalid URL", func(t *testing.T) {
		t.Parallel()
		is := assert.New(t)

		_, err := ParseImageURI("://invalid-url")
		is.Error(err)
	})
}
