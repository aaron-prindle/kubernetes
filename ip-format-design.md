# Declarative IP Format Tags One-Pager

## TL;DR

This branch adds two Kubernetes-specific declarative validation formats:

- `+k8s:format=k8s-ip`
- `+k8s:format=k8s-ip-sloppy`

The split is intentional. Kubernetes has two different IP validation contracts today:

- new fields should use strict, canonical IP validation;
- legacy fields need the same compatibility behavior as handwritten `IsValidIPForLegacyField`.

`k8s-ip` is for strict fields. `k8s-ip-sloppy` is for legacy fields and is feature-gate-aware through `StrictIPCIDRValidation`.

The important migration boundary is list ratcheting: `k8s-ip-sloppy` is safe for scalars and for list elements where validation-gen can correlate old and new elements, such as `listType=map` keyed by IP. It is not yet safe for atomic legacy IP lists that rely on old-membership ratcheting.

## Background

Kubernetes currently has handwritten IP validation in:

- `staging/src/k8s.io/apimachinery/pkg/util/validation/ip.go`
- wrappers and callsites in `pkg/apis/core/validation/validation.go`
- networking validation callsites such as `pkg/apis/networking/validation/validation.go`

The handwritten API distinguishes strict and legacy behavior:

- `IsValidIP`: strict parse, rejects ambiguous forms, requires canonical string form.
- `IsValidIPForLegacyField`: legacy-compatible parse, optionally strictens based on `StrictIPCIDRValidation`, and accepts specific old values for ratcheting.

The existing OpenAPI formats do not give us this contract. OpenAPI has `ipv4`, `ipv6`, and `cidr`, but no generic Kubernetes `ip` format. Also, OpenAPI `ipv4` currently uses sloppy parsing, which is not the same as Kubernetes strict `IsValidIP`.

Declarative validation therefore needs Kubernetes-native formats rather than reusing OpenAPI format names.

## Goals

- Add declarative validation support for strict IP fields.
- Add declarative validation support for legacy IP fields.
- Preserve handwritten validation behavior, including feature-gated strictness.
- Preserve error origins:
  - strict IP: `format=ip-strict`
  - legacy IP: `format=ip-sloppy`
- Provide a safe first migration path.
- Explicitly document where the tags are not yet sufficient.

## Non-Goals

- Do not migrate every IP field in one PR.
- Do not change API list semantics to get validation ratcheting.
- Do not make atomic lists behave like sets or maps.
- Do not solve CIDR validation in this initial IP-focused change.
- Do not add declarative warning support for `GetWarningsForIP`.
- Do not claim OpenAPI `ipv4`/`ipv6` parity.

## Proposed Tags

### `+k8s:format=k8s-ip`

Strict IP validation.

Implementation:

- generator registration: `staging/src/k8s.io/code-generator/cmd/validation-gen/validators/format.go`
- runtime validator: `staging/src/k8s.io/apimachinery/pkg/api/validate/strfmt.go`
- underlying helper: `validation.IsValidIP`

Semantics:

- requires a valid IPv4 or IPv6 address;
- rejects IPv4 octets with leading zeroes;
- rejects IPv4-mapped IPv6 addresses;
- requires canonical string form;
- is not feature-gated;
- does not need legacy ratcheting.

Use this for new fields or existing fields that already use handwritten `IsValidIP`.

Example migration in this branch:

- `IPAddress.metadata.name` in `networking.k8s.io/v1`
- `IPAddress.metadata.name` in `networking.k8s.io/v1beta1`

Why this field works:

- handwritten validation already requires the name to be a canonical IP;
- no legacy sloppy ratcheting is involved;
- the field is scalar metadata name validation.

### `+k8s:format=k8s-ip-sloppy`

Legacy IP validation.

Implementation:

- generator registration: `staging/src/k8s.io/code-generator/cmd/validation-gen/validators/format.go`
- runtime validator: `staging/src/k8s.io/apimachinery/pkg/api/validate/strfmt.go`
- underlying helper: `validation.IsValidIPForLegacyField`

Semantics:

- uses legacy-compatible IP parsing;
- when `StrictIPCIDRValidation` is disabled, allows historical ambiguous forms;
- when `StrictIPCIDRValidation` is enabled, rejects leading-zero IPv4 and IPv4-mapped IPv6;
- does not require canonical IPv6 form, matching `IsValidIPForLegacyField`;
- preserves `format=ip-sloppy`;
- handles scalar same-value update ratcheting through the old scalar value passed by generated validation.

Use this for fields that currently use handwritten `IsValidIPForLegacyField`.

Example migration in this branch:

- `PodStatus.PodIPs[*].IP`

Why this field works:

- handwritten validation uses `IsValidIPForLegacyField`;
- `PodStatus.PodIPs` is `listType=map` keyed by `ip`;
- validation-gen can correlate old and new list elements by that key;
- moved old legacy IP strings are still recognized as old values.

## Feature Gate Handling

Handwritten core validation currently does:

```go
validation.IsValidIPForLegacyField(fldPath, value,
    utilfeature.DefaultFeatureGate.Enabled(features.StrictIPCIDRValidation),
    validOldIPs)
```

The declarative runtime validator does not read the global feature gate directly. Instead, it checks the operation option:

```go
op.HasOption("StrictIPCIDRValidation")
```

The strategy is responsible for populating that option in `DeclarativeValidationConfig`.

Why this choice:

- matches existing declarative validation architecture;
- keeps runtime validators deterministic from `operation.Operation`;
- allows resource strategies to decide effective feature state;
- avoids reaching into global feature-gate state from generated validation.

## Rationale For Two Tags

A single `k8s-ip` tag would be ambiguous.

If `k8s-ip` meant strict validation, it would not match legacy fields like `PodIPs`, `ClusterIPs`, and `ExternalIPs`.

If `k8s-ip` meant sloppy validation, it would not match new strict fields like `IPAddress.metadata.name`.

Splitting the tags mirrors the existing helper API:

| Declarative tag | Runtime validator | Handwritten helper | Intended use |
|---|---|---|---|
| `k8s-ip` | `validate.IP` | `validation.IsValidIP` | strict/new fields |
| `k8s-ip-sloppy` | `validate.IPSloppy` | `validation.IsValidIPForLegacyField` | legacy fields |

This makes the migration explicit at every callsite.

## Rationale Against `+k8s:ifEnabled` / `+k8s:ifDisabled`

One possible design was composing existing tags:

- strict branch when `StrictIPCIDRValidation` is enabled;
- sloppy branch when disabled.

That does not match handwritten behavior.

The strict branch would likely use `IsValidIP`, which requires canonical form. But `IsValidIPForLegacyField` with strict validation enabled does not require canonical IPv6 form. It only tightens ambiguous forms such as leading-zero IPv4 and IPv4-mapped IPv6.

Therefore, `IPSloppy` needs its own feature-gate-aware implementation rather than being composed from `IP`.

## Rationale Against OpenAPI Formats

OpenAPI `format: ipv4` and `format: ipv6` are not equivalent to these tags.

Important differences:

- OpenAPI has separate `ipv4` and `ipv6`, not a generic Kubernetes IP format.
- OpenAPI `ipv4` uses sloppy parsing today.
- Kubernetes strict IP validation requires canonical form.
- Kubernetes legacy IP validation has feature-gated strictening and ratcheting.

These tags are Kubernetes API validation contracts, not OpenAPI schema format aliases.

## Ratcheting Model

### Handwritten legacy behavior

`IsValidIPForLegacyField` accepts old values through a caller-provided list:

```go
if slices.Contains(validOldIPs, value) {
    return nil
}
```

That means old-membership ratcheting is by value across the old parent list, not by index.

Example:

```go
old: ["010.000.000.001"]
new: ["10.0.0.2", "010.000.000.001"]
```

Handwritten validation allows the moved legacy IP because it still appears in the old list.

### Declarative built-in behavior

Declarative validation uses old/new correlation:

- scalar field: old scalar value is available;
- `listType=map`: old item is found by key;
- `listType=set`: old item is found by whole-element equality;
- `listType=atomic`: no per-element old correlation is available.

This is implemented in `validation-gen/validators/each.go` and the runtime helper `validate.EachSliceVal`.

### Safe and unsafe cases

| Shape | `k8s-ip` | `k8s-ip-sloppy` | Notes |
|---|---:|---:|---|
| scalar string | yes | yes | scalar old value is available |
| metadata subfield, e.g. `metadata.name` | yes | usually no | strict names are good candidates |
| `listType=map` keyed by IP | yes | yes | old/new elements correlate by key |
| `listType=set` of strings | yes | maybe | works only if set semantics match handwritten behavior |
| `listType=atomic` scalar list | yes for strict fields | not yet for legacy ratcheting | old-membership primitive needed |
| atomic struct list with IP subfield | yes for strict fields | not yet for legacy ratcheting | projected old-membership primitive needed |

## Why Atomic Legacy Lists Are Not Migrated Yet

Atomic legacy IP lists often have handwritten validation that passes all old IPs as `validOldIPs`.

Examples to treat carefully:

- `Service.Spec.ClusterIPs`
- `Service.Spec.ExternalIPs`
- `Pod.Status.HostIPs`
- CIDR analogues such as `LoadBalancerSourceRanges`

Generated `EachSliceVal` for atomic lists currently receives no match function. If an old legacy IP moves because of insertion, deletion, or reordering, declarative validation treats it as new and may reject it.

This would be a behavior change.

We should not solve this by changing list type or adding `+k8s:unique` unless the handwritten validation already has identical uniqueness behavior. `+k8s:unique` can affect validation identity, but it also emits uniqueness errors. That is broader than the ratcheting behavior we need.

The missing primitive is validation-local old-membership correlation, for example a future tag like:

```go
// +k8s:eachVal=+k8s:ratchetBy(self)=+k8s:format=k8s-ip-sloppy
```

or for struct lists:

```go
// +k8s:eachVal=+k8s:ratchetBy(ip)=+k8s:subfield(ip)=+k8s:format=k8s-ip-sloppy
```

The key property is that this should affect only validation ratcheting, not OpenAPI list semantics, merge behavior, apply ownership, or uniqueness validation.

## Current Proof Migrations

### `PodStatus.PodIPs[*].IP`

Files:

- `staging/src/k8s.io/api/core/v1/types.go`
- `pkg/apis/core/v1/zz_generated.validations.go`
- `pkg/apis/core/validation/validation.go`
- `pkg/registry/core/pod/strategy.go`
- `pkg/registry/core/pod/declarative_validation_test.go`

Behavior:

- uses `+k8s:format=k8s-ip-sloppy`;
- strategy passes `StrictIPCIDRValidation` as a declarative validation option;
- handwritten `validatePodIPs` error is marked covered by declarative validation;
- tests cover strict rejection, strict-off legacy acceptance, unchanged legacy ratcheting, deletion, canonical replacement, different legacy rejection, and moved legacy ratcheting by list-map key.

Known caveat:

- generated Pod validation is registered for `/`, `/status`, and `/ephemeralcontainers` because the type supports those subresources;
- the handwritten `PodIPs` validation corresponds specifically to status validation;
- normal strategy field wiping should make this inert outside `/status`, but this should be called out in the PR.

### `IPAddress.metadata.name`

Files:

- `staging/src/k8s.io/api/networking/v1/types.go`
- `staging/src/k8s.io/api/networking/v1beta1/types.go`
- `pkg/apis/networking/v1/zz_generated.validations.go`
- `pkg/apis/networking/v1beta1/zz_generated.validations.go`
- `pkg/apis/networking/validation/validation.go`
- `pkg/registry/networking/ipaddress/declarative_validation_test.go`

Behavior:

- uses `+k8s:format=k8s-ip`;
- matches the API doc requirement that the object name is a canonical IP string;
- handwritten name validation now uses `ValidateObjectMetaWithOpts` so the strict IP error can preserve origin and be marked covered;
- tests cover non-canonical IPv4 and IPv6 names.

## Migration Guidance

Use `k8s-ip` when:

- the handwritten code uses `IsValidIP`;
- the field is strict and canonical;
- the field is new or does not require legacy compatibility.

Use `k8s-ip-sloppy` when:

- the handwritten code uses `IsValidIPForLegacyField`;
- the field is scalar; or
- the field is in a list where validation-gen already correlates old and new items.

Do not use `k8s-ip-sloppy` yet when:

- the field is an atomic list element; and
- handwritten validation passes a parent old-value list for ratcheting.

For those fields, wait for validation-local old-membership correlation.

## Testing Strategy

Required test types:

- runtime validator unit tests for strict and sloppy validators;
- validation-gen output fixtures for both tags;
- migration equivalence tests against registry strategies;
- explicit ratcheting cases for legacy fields.

Ratcheting test cases should include:

- create with valid canonical IP;
- create/update with new legacy IP rejected under `StrictIPCIDRValidation`;
- legacy IP accepted when strict validation is disabled;
- unchanged legacy IP accepted on update;
- moved legacy IP accepted when list correlation supports it;
- different new legacy IP rejected;
- replacement with canonical equivalent accepted.

## Risks

| Risk | Mitigation |
|---|---|
| Atomic-list legacy fields get migrated too early | Document boundary and avoid those fields until old-membership primitive exists |
| `k8s-ip-sloppy` accidentally uses strict canonical validation | Dedicated `IPSloppy` implementation, not `ifEnabled` composition with `IP` |
| Feature gate state differs from handwritten validation | Strategy passes `StrictIPCIDRValidation` through `Operation.Options` |
| OpenAPI format behavior is assumed equivalent | Use Kubernetes-specific tag names and document OpenAPI mismatch |
| Generated uniqueness errors appear unexpectedly | Do not use `+k8s:unique` as a ratcheting workaround unless handwritten uniqueness matches |
| Pod subresource registration is broader than handwritten callsite | Document PR caveat and rely on strategy field wiping for current proof migration |

## Open Follow-Ups

- Add validation-local old-membership ratcheting for atomic lists.
- Add CIDR counterparts:
  - `k8s-cidr`
  - `k8s-cidr-sloppy`
  - possibly `k8s-interface-address`
- Add declarative warning support for `GetWarningsForIP` / CIDR warnings.
- Decide whether field-level subresource scoping is needed for migrations like Pod status fields.
- Revisit duplicate and empty-value behavior for future list migrations before marking handwritten checks covered.

