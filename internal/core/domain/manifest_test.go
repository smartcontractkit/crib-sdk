package domain_test

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/filehandler"
	"github.com/smartcontractkit/crib-sdk/internal/core/domain"
)

func TestUnmarshalDocument(t *testing.T) {
	t.Parallel()
	is := assert.New(t)
	must := require.New(t)

	ctx := t.Context()

	fh, err := filehandler.New(ctx, "testdata")
	must.NoError(err)
	raw, err := fh.ReadFile("multi_obj.yaml")
	must.NoError(err)
	must.NotNil(raw)

	for manifest, err := range domain.UnmarshalDocument(raw) {
		if !is.NoError(err) {
			continue
		}
		snaps.MatchYAML(t, manifest)
	}
}
