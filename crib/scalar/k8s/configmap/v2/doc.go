// Package configmapv2 provides a Kubernetes ConfigMap component for Crib SDK.
//
// This package provides both a Scalar API and a Composite Mapper API for working with Kubernetes ConfigMaps.
// To use as a Scalar:
//
//	import (
//		"github.com/smartcontractkit/crib-sdk/crib"
//		configmap "github.com/smartcontractkit/crib-sdk/crib/scalar/k8s/configmap/v2"
//	)
//
//	func Plan() *crib.Plan {
//		return crib.NewPlan("my-plan",
//			crib.ComponentSet(
//				configmap.Scalar(
//					"my-configmap", // The name of the ConfigMap.
//					configmap.WithNamespace("my-namespace"), // Optional namespace.
//					configmap.WithData(map[string]string{"key": "value"}), // Data for the ConfigMap.
//					configmap.WithAppName("my-app"), // Required app name.
//					configmap.WithAppInstance("my-instance"), // Required app instance.
//				),
//			),
//		)
//	}
//
// To utilize the Composite Mapper API a Scalar Component must implement the IConfigMap interface:
//
//		type MyScalar struct {
//	    	// Other fields...
//		}
//
//		// Other required methods like String() and Apply().
//
//		func (s *MyScalar) ConfigMap() *configmap.Component {
//			return &configmap.Component{
//				// Fill in the required fields for the ConfigMap.
//			}
//		}
//
// The Component can then be used in a Composite Mapper as follows:
//
//	import (
//		"github.com/smartcontractkit/crib-sdk/crib"
//		configmap "github.com/smartcontractkit/crib-sdk/crib/scalar
//		myscalar "myscalar/package/path"
//	)
//
//	func Plan() *crib.Plan {
//		return crib.NewPlan("my-composite-plan",
//			crib.ComponentSet(
//				crib.NewComposite(
//					myscalar.Scalar(),
//					myscalar.Scalar(), // As an example, multiple Scalars can be used.
//
//					// The ConfigMap Mapper will automatically find all Scalars that implement IConfigMap
//					// and apply any supplied ConfigMapOpts to them.
//					// If supplied, these will be applied to all ConfigMaps before the manifest is generated.
//					configmap.Components(),
//				)
//			),
//		)
//	}
package configmapv2
