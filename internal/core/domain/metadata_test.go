package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMetadata(t *testing.T) {
	tests := []struct {
		name    string
		props   *DefaultResourceMetadataProps
		wantErr bool
	}{
		{
			name: "basic metadata with minimal props",
			props: &DefaultResourceMetadataProps{
				AppName:      "test-app",
				ResourceName: "test-app-deployment",
				Namespace:    "test-namespace",
				AppInstance:  "test-instance",
			},
			wantErr: false,
		},
		{
			name: "metadata with custom labels",
			props: &DefaultResourceMetadataProps{
				AppName:      "test-app",
				ResourceName: "test-app-deployment",
				Namespace:    "test-namespace",
				AppInstance:  "test-instance",
				ExtraLabels: Labels{
					"custom-label":              "custom-value",
					"app.kubernetes.io/version": "1.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "metadata with annotations",
			props: &DefaultResourceMetadataProps{
				AppName:      "test-app",
				ResourceName: "test-app-deployment",
				Namespace:    "test-namespace",
				AppInstance:  "test-instance",
				ExtraAnnotations: Annotations{
					"example.com/annotation": "annotation-value",
				},
			},
			wantErr: false,
		},
		{
			name: "metadata with custom labels and annotations",
			props: &DefaultResourceMetadataProps{
				AppName:      "test-app",
				ResourceName: "test-resource",
				Namespace:    "test-namespace",
				AppInstance:  "test-instance",
				ExtraLabels: Labels{
					"custom-label": "custom-value",
				},
				ExtraAnnotations: Annotations{
					"example.com/annotation": "annotation-value",
				},
			},
			wantErr: false,
		},
		{
			name: "validation failure - missing required fields",
			props: &DefaultResourceMetadataProps{
				// Missing AppName, Namespace, and AppInstance
				ResourceName: "test-resource",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			must := require.New(t)
			is := assert.New(t)

			metadataFactory, err := NewMetadataFactory(tt.props)
			// Check error case
			if tt.wantErr {
				must.Error(err, "should return an error when validation fails")
				must.Nil(metadataFactory, "metadataFactory should be nil when validation fails")
				return
			}

			meta := metadataFactory.K8sResourceMetadata()

			// Continue with normal test cases
			must.NoError(err, "should not return an error for valid props")
			must.NotNil(meta, "metadata should not be nil")

			// Check Name and Namespace
			is.Equal(tt.props.ResourceName, *meta.Name, "metadata name should match")
			is.Equal(tt.props.Namespace, *meta.Namespace, "metadata namespace should match")

			// Check default labels
			is.NotNil(meta.Labels, "labels should not be nil")
			checkMetadataField(must, *meta.Labels, "app.kubernetes.io/instance", tt.props.AppInstance)
			checkMetadataField(must, *meta.Labels, "app.kubernetes.io/name", tt.props.AppName)

			// Check custom labels
			for k, v := range tt.props.ExtraLabels {
				checkMetadataField(must, *meta.Labels, k, v)
			}

			// Check annotations
			if len(tt.props.ExtraAnnotations) > 0 {
				must.NotNil(meta.Annotations, "annotations should not be nil when extra annotations are provided")
				for k, v := range tt.props.ExtraAnnotations {
					checkMetadataField(must, *meta.Annotations, k, v)
				}
			}

			// Check that the number of labels matches what's expected
			expectedLabelsCount := 2 + len(tt.props.ExtraLabels) // 2 for the default labels
			is.Equal(expectedLabelsCount, len(*meta.Labels), "number of labels should match expected count")

			// Check that the number of annotations matches what's expected
			if meta.Annotations != nil {
				is.Equal(len(tt.props.ExtraAnnotations), len(*meta.Annotations), "number of annotations should match expected count")
			} else {
				is.Empty(tt.props.ExtraAnnotations, "annotations should be nil when no extra annotations are provided")
			}
		})
	}
}

func TestDefaultSelectorLabels(t *testing.T) {
	tests := []struct {
		name  string
		props *DefaultResourceMetadataProps
	}{
		{
			name: "selector with complete props",
			props: &DefaultResourceMetadataProps{
				AppName:      "complex-app",
				AppInstance:  "complex-instance",
				Namespace:    "test-namespace",
				ResourceName: "test-resource",
				ExtraLabels: Labels{
					"custom-label": "custom-value",
				},
				ExtraAnnotations: Annotations{
					"example.com/annotation": "annotation-value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := assert.New(t)
			must := require.New(t)

			metadataFactory, err := NewMetadataFactory(tt.props)
			must.NoError(err)

			selectorLabels := metadataFactory.SelectorLabels()

			// Verify the result is not nil
			must.NotNil(selectorLabels, "selector labels should not be nil")

			// Verify the map itself is not nil
			must.NotNil(*selectorLabels, "selector labels map should not be nil")

			// Check that the expected keys exist with proper values using the helper function
			checkMetadataField(must, *selectorLabels, "app.kubernetes.io/instance", tt.props.AppInstance)
			checkMetadataField(must, *selectorLabels, "app.kubernetes.io/name", tt.props.AppName)

			// Verify no extra labels are included (only the two standard ones)
			is.Equal(2, len(*selectorLabels), "selector labels should only contain the two standard labels")
		})
	}
}

// checkMetadataField can validate if labels or annotations contain a required key and if it matches the expected value
func checkMetadataField(must *require.Assertions, labels map[string]*string, key, expectedValue string) {
	must.Contains(labels, key, "Label %q should exist", key)
	must.Equal(expectedValue, *labels[key], "label %q should exist", key)
}
