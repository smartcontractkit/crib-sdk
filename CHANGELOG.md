# Changelog

## 0.5.0 (2025-08-12)

## What's Changed
* ci: Add method for checking conventional commit status by @polds in https://github.com/smartcontractkit/crib-sdk/pull/16
* build(deps): bump github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2 from 2.70.2 to 2.70.3 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/21
* Add task to install Helm secrets plugin by @njegosrailic in https://github.com/smartcontractkit/crib-sdk/pull/20
* feat(composite): implement Composite API with automatic dependency resolution by @polds in https://github.com/smartcontractkit/crib-sdk/pull/23
* fix: Don’t provide commit_msg to GITHUB_OUTPUT by @polds in https://github.com/smartcontractkit/crib-sdk/pull/25
* feat: use GitHub’s API to generate release notes by @polds in https://github.com/smartcontractkit/crib-sdk/pull/26
* build(deps): bump github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2 from 2.70.3 to 2.70.4 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/28
* fix: Remove go.work and ignore it by @polds in https://github.com/smartcontractkit/crib-sdk/pull/34
* build(deps): bump github.com/aws/jsii-runtime-go from 1.112.0 to 1.113.0 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/27
* build(deps): bump github.com/gkampitakis/go-snaps from 0.5.13 to 0.5.14 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/29
* build(deps): bump github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2 from 2.70.4 to 2.70.7 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/31
* build(deps): bump github.com/cdk8s-team/cdk8s-plus-go/cdk8splus30/v2 from 2.4.8 to 2.4.10 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/32
* build(deps): bump actions/checkout from 4 to 5 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/36
* build(deps): bump golang.org/x/mod from 0.26.0 to 0.27.0 by @dependabot[bot] in https://github.com/smartcontractkit/crib-sdk/pull/33
* fix: Update the auto-ready workflow to hopefully not fail by @polds in https://github.com/smartcontractkit/crib-sdk/pull/35
* fix: Update how ComponentNames are derived to be more correct by @polds in https://github.com/smartcontractkit/crib-sdk/pull/37
* feat: initial configmap/v2 with support for new Composite API by @polds in https://github.com/smartcontractkit/crib-sdk/pull/30

## New Contributors
* @njegosrailic made their first contribution in https://github.com/smartcontractkit/crib-sdk/pull/20

**Full Changelog**: https://github.com/smartcontractkit/crib-sdk/compare/v0.4.0...v0.5.0

## [0.4.0](https://github.com/smartcontractkit/crib-sdk/compare/v0.3.0...v0.4.0) (2025-07-22)


### Features

* Option to quit existing telepresence session ([#18](https://github.com/smartcontractkit/crib-sdk/issues/18)) ([8f3d5ba](https://github.com/smartcontractkit/crib-sdk/commit/8f3d5ba49ae3ac9ea523f369fa69314598b721f5))


### Bug Fixes

* Address go:generate flakiness by locally vendoring the used templates ([#15](https://github.com/smartcontractkit/crib-sdk/issues/15)) ([d115a6c](https://github.com/smartcontractkit/crib-sdk/commit/d115a6cd49b0ca53eb16f48f301492bd5bddd2bb))

## [0.3.0](https://github.com/smartcontractkit/crib-sdk/compare/v0.2.0...v0.3.0) (2025-07-21)


### Features

* trigger release ([#12](https://github.com/smartcontractkit/crib-sdk/issues/12)) ([2388bb8](https://github.com/smartcontractkit/crib-sdk/commit/2388bb80c1df89dfa21e76c4f69c5bf2990d67e0))

## [0.2.0](https://github.com/smartcontractkit/crib-sdk/compare/v0.1.0...v0.2.0) (2025-07-18)


### Features

* new apis for external scalars / composites ([#2](https://github.com/smartcontractkit/crib-sdk/issues/2)) ([11e0092](https://github.com/smartcontractkit/crib-sdk/commit/11e009265e26e763a33d53d4df6dda198fc1c12e))
* nodeset/v1: implement waiting for nodes ([cebdb7f](https://github.com/smartcontractkit/crib-sdk/commit/cebdb7fe5c8e57433e7e5e8df8f57d56d19f614c))
* **nodeset/v1:** implement waiting for nodes ([#3](https://github.com/smartcontractkit/crib-sdk/issues/3)) ([cebdb7f](https://github.com/smartcontractkit/crib-sdk/commit/cebdb7fe5c8e57433e7e5e8df8f57d56d19f614c))

## 0.1.0 (2025-07-17)

### Features

* Initial open source release ([7a4ff38](https://github.com/smartcontractkit/crib-sdk/commit/7a4ff38e44b518c21b50076b3ca82ee1a84dd3d7))
