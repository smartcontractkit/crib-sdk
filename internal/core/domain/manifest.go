package domain

import (
	"errors"
	"io"
	"iter"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/crib-sdk/internal/adapter/mempools"
)

const (
	CribAPIVersion  = "crib.smartcontract.com/v1alpha1"
	ClientSideApply = "ClientSideApply"

	DefaultNamespace = "default"
)

type (
	unmarshalableManifest interface {
		Manifest | ClientSideApplyManifest
	}

	// Manifest represents a basic Kubernetes manifest.
	Manifest struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}

	// GenericManifest represents a manifest that can hold any kind of data.
	GenericManifest map[string]any
)

// UnmarshalManifest unmarshals a YAML byte slice into a Manifest struct.
func UnmarshalManifest[T unmarshalableManifest](data []byte) (*T, error) {
	var t T
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// UnmarshalDocument reads and parses an incoming document which may contain zero or more manifests.
// It unmarshals each into a GenericManifest and returns an iterable slice of GenericManifest.
// Use:
//
//	raw := nil // Get your raw YAML data here.
//	for manifest, err := range UnmarshalDocument(raw) {
//		if err != nil { // Handle error }
//		_ = manifest // Manifest is a GenericManifest (map[string]any)
//	}
func UnmarshalDocument(raw []byte) iter.Seq2[GenericManifest, error] {
	buf, reset := mempools.BytesBuffer.Get()
	defer reset()

	buf.Write(raw)
	dec := yaml.NewDecoder(buf)
	return func(yield func(GenericManifest, error) bool) {
		for {
			var doc GenericManifest
			err := dec.Decode(&doc)
			if errors.Is(err, io.EOF) {
				break // End of file reached.
			}
			if !yield(doc, err) {
				return
			}
		}
	}
}
