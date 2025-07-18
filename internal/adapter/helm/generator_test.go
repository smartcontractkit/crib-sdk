package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_normalizePackageName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"single word", "foo", "foo"},
		{"with dashes", "foo-bar", "foobar"},
		{"with underscores", "foo_bar", "foobar"},
		{"with mixed case", "FooBar", "foobar"},
		{"starts with number", "123foo", "pkg_123foo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &Generator{
				TemplateOpts: TemplateOpts{
					Defaults: &Defaults{
						Release: Release{
							ReleaseName: tc.input,
						},
					},
				},
			}

			assert.Equal(t, g.normalizePackageName(), tc.want)
		})
	}
}
