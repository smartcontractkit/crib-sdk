# Examples

This directory contains examples on how to utilize the CRIB-SDK.

```golang
package main

import (
   "context"

   "github.com/smartcontractkit/crib-sdk/crib"
   nodesetv1 "github.com/smartcontractkit/crib-sdk/service/nodeset/v1"
   deploymentv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/deployment/v1"
   configmapv1 "github.com/smartcontractkit/crib-sdk/crib/scalar/configmap/v1
   ccipv1 "github.com/smartcontractkit/crib-sdk/contrib/ccip/v1"
)

func main() {
    ctx := context.Background()
    cfg := crib.NewConfig(
       crib.WithContext(ctx),
    )

    nginx := deploymentv1.New("nginx", "nginx:1.25", 1)
    nodeset := nodesetv1.New("ocr", "chainlink/node:1.6")
    // Example of a lazily initialized resource based on previous input.
    configmap := configmapv1.ConfigMap{
        Name: "my-configmap",
        ValuesFunc: func() map[string]any{
            return map[string]any{
                "nginx-addr": nginx.String(),
                "nodeset-version": nodeset.Values().Version().String(),
            }
        },
        WaitFor: []runtime.Component{
            nginx,
            nodeset,
        },
    }

    plan := crib.NewPlan(
        cfg,
        crib.Namespace("my-namespace"),
        crib.ComponentSet(
            nginx,
            nodeset,
            configmap,
            // Add a loadtest.
            crib.NewPlan(
                cfg,
                crib.ComponentSet(
                   ccipv1.LoadTest{
                      ValuesFunc: func() map[string]any{
                         "endpoint": configmap.Values()["nginx-addr"],
                      }
                      WaitFor: []crib.Component{configmap},
                   ),
               ),
           ),
       ),
    )

    res, err := plan.Execute(ctx)
}

```
