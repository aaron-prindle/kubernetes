# Declarative IP Validation Implementation Plan

## Goal

Enable declarative validation for Kubernetes IP formats while preserving the behavior of existing handwritten validation, including `StrictIPCIDRValidation` and legacy old-value ratcheting.

The implementation must not reject valid updates to existing objects that contain formerly accepted IP strings, such as IPv4 octets with leading zeroes or IPv4-mapped IPv6 addresses.

## Implementation Status

- [x] Phase 1: runtime `IP` and `IPSloppy` validators plus unit tests.
- [x] Phase 2: validation-gen wiring, docs, and output tests for `k8s-ip` and `k8s-ip-sloppy`.
- [ ] Phase 3: feature-gate option plumbing in real API strategies.
  - [x] Pod strategy now passes `StrictIPCIDRValidation` into declarative validation.
- [ ] Phase 4: projected old-membership ratcheting primitive for atomic lists.
- [ ] Phase 5+: field classification, proof migrations, atomic-list migrations, warnings, and CIDR follow-up.
  - [x] Proof migration attempted for `PodStatus.PodIPs[*].IP`, a `listType=map` field where key-based declarative ratcheting is sufficient.

## Key Decisions

1. Use two declarative formats, not one.
   - `+k8s:format=k8s-ip` means strict, canonical IP validation equivalent to `validation.IsValidIP`.
   - `+k8s:format=k8s-ip-sloppy` means legacy IP validation equivalent to `validation.IsValidIPForLegacyField`.

2. Make `IPSloppy` internally gate-aware.
   - It should inspect `op.HasOption("StrictIPCIDRValidation")`.
   - Do not compose sloppy behavior with `+k8s:ifEnabled(...)=+k8s:format=k8s-ip`, because strict `IP` enforces canonical form and handwritten legacy validation does not.

3. Do not rely on built-in declarative ratcheting for atomic lists.
   - `listType=map` and `listType=set` can correlate old values.
   - `listType=atomic` cannot correlate old elements.
   - Handwritten legacy IP validation often accepts any exact old IP string from the old parent list, which built-in `eachVal` does not model for atomic lists.

4. Add an explicit old-membership ratcheting primitive before migrating atomic-list IP fields.
   - The primitive must support both scalar lists, such as `[]string`, and struct lists, such as `[]LoadBalancerIngress`, where the ratcheted value is a projected field like `.IP`.

## Current State

- `+k8s:format=k8s-ip` is documented in `staging/src/k8s.io/code-generator/cmd/validation-gen/validators/format.go`, but its generator mapping is commented out.
- The commented mapping points to a non-existent runtime validator, `validate.IPSloppy`.
- Current docs for `k8s-ip` describe sloppy behavior. Since the format is not enabled yet, update the docs before exposing it.
- Handwritten legacy IP validation is centralized in `staging/src/k8s.io/apimachinery/pkg/util/validation/ip.go`.
- The apiserver-facing feature-gated wrapper is in `pkg/apis/core/validation/validation.go`.
- Declarative validation options are carried through `rest.DeclarativeValidationConfig.Options` into `operation.Operation.Options`.

## Phase 1: Runtime Validators

Add validators to:

`staging/src/k8s.io/apimachinery/pkg/api/validate/strfmt.go`

### `IP`

Shape:

```go
func IP[T ~string](ctx context.Context, op operation.Operation, fldPath *field.Path, value, oldValue *T) field.ErrorList
```

Behavior:

- Return nil for nil values.
- Validate with `validation.IsValidIP`.
- This is strict and canonical.
- It is gate-blind.
- It should preserve the handwritten strict origin, currently `format=ip-strict`.

Use this for new API fields that should not accept legacy sloppy syntax.

### `IPSloppy`

Shape:

```go
func IPSloppy[T ~string](ctx context.Context, op operation.Operation, fldPath *field.Path, value, oldValue *T) field.ErrorList
```

Behavior:

- Return nil for nil values.
- On update, if `oldValue != nil` and `*value == *oldValue`, return nil.
- Determine strictness from `op.HasOption("StrictIPCIDRValidation")`.
- Validate with `validation.IsValidIPForLegacyField(fldPath, string(*value), strict, nil)`.
- Do not enforce canonical form.
- It should preserve the handwritten legacy origin, currently `format=ip-sloppy`.

Do not import `pkg/features` or `utilfeature` into apimachinery.

### Unit Tests

Add table-driven tests in:

`staging/src/k8s.io/apimachinery/pkg/api/validate/strfmt_test.go`

Cover:

- canonical IPv4;
- canonical IPv6;
- empty string;
- garbage string;
- IPv4 with leading zeroes;
- IPv4-mapped IPv6;
- non-canonical IPv6;
- update retaining an old bad value;
- `IPSloppy` with `StrictIPCIDRValidation` option absent and present;
- origin strings.

## Phase 2: Generator Wiring

Update:

`staging/src/k8s.io/code-generator/cmd/validation-gen/validators/format.go`

Add runtime symbols:

```go
ipValidator       = types.Name{Package: libValidationPkg, Name: "IP"}
ipSloppyValidator = types.Name{Package: libValidationPkg, Name: "IPSloppy"}
```

Add switch cases:

```go
case "k8s-ip":
    return Function(formatTagName, DefaultFlags, ipValidator), nil
case "k8s-ip-sloppy":
    return Function(formatTagName, DefaultFlags, ipSloppyValidator), nil
```

Update docs:

- `k8s-ip`: strict IPv4 or IPv6 address in canonical Kubernetes form.
- `k8s-ip-sloppy`: legacy IPv4 or IPv6 address; may allow sloppy syntax depending on `StrictIPCIDRValidation`; intended only for legacy fields.

Add validation-gen output tests:

- scalar `string`;
- `*string`;
- string typedef;
- map key if supported;
- map value;
- `+k8s:eachVal` on `listType=set`;
- `+k8s:eachVal` on `listType=map`.

Do not use atomic-list output tests as proof of ratcheting equivalence. Atomic lists require the explicit primitive in Phase 4.

## Phase 3: Feature-Gate Option Plumbing

Declarative validators receive feature-like options through:

- `rest.DeclarativeValidationConfig.Options`;
- `runtime.Scheme.Validate` / `ValidateUpdate`;
- `operation.Operation.Options`;
- `op.HasOption(...)`.

For every resource strategy that uses `k8s-ip-sloppy`, override `DeclarativeValidationConfig` and append `StrictIPCIDRValidation` when the feature gate is enabled.

Example:

```go
func (s strategy) DeclarativeValidationConfig(ctx context.Context, obj, oldObj runtime.Object) rest.DeclarativeValidationConfig {
    cfg := s.DeclarativeValidation.DeclarativeValidationConfig(ctx, obj, oldObj)
    if utilfeature.DefaultFeatureGate.Enabled(features.StrictIPCIDRValidation) {
        cfg.Options = append(cfg.Options, string(features.StrictIPCIDRValidation))
    }
    return cfg
}
```

If the strategy already has custom config, append to its existing options rather than replacing them.

Add tests or generated validation fixtures proving `op.HasOption("StrictIPCIDRValidation")` reaches `IPSloppy`.

## Phase 4: Old-Membership Ratcheting For Atomic Lists

Built-in declarative ratcheting is not enough for atomic lists.

The generator currently behaves as follows:

- `listType=map`: correlate old/new elements by list-map key.
- `listType=set`: correlate old/new elements by full value.
- `listType=atomic`: no old element correlation.

Handwritten legacy IP validation often does this instead:

```go
if slices.Contains(validOldIPs, value) {
    return nil
}
```

That means any exact old IP string from the old parent list remains valid, even if the list changed.

### Required Primitive

Add a narrow helper in `staging/src/k8s.io/apimachinery/pkg/api/validate` that supports projected old-membership ratcheting.

Suggested general shape:

```go
func EachSliceValRatchetedBy[T any, K comparable](
    ctx context.Context,
    op operation.Operation,
    fldPath *field.Path,
    newSlice, oldSlice []T,
    key func(T) K,
    itemPath func(*field.Path, int) *field.Path,
    validator func(context.Context, operation.Operation, *field.Path, K) field.ErrorList,
) field.ErrorList
```

Behavior:

- On update, compute old keys from `oldSlice`.
- For each new item:
  - compute its key;
  - if key exists in old keys, skip validation;
  - otherwise validate the key.
- On create, validate every key.

This supports:

- scalar lists with identity key, such as `[]string`;
- struct lists with projected key, such as `[]LoadBalancerIngress` using `.IP`;
- future CIDR equivalents.

Alternative narrower names are fine, for example `EachSliceValIPSloppyBy`, but avoid a helper that only works for `[]string`; load-balancer ingress lists need projected IP membership.

Do not change the semantics of existing `EachSliceVal` for atomic lists.

### Tests For The Primitive

Add tests covering:

- old bad scalar IP retained while a new valid IP is added;
- old bad scalar IP retained after reorder;
- new bad scalar IP rejected;
- old bad IP removed;
- bad IP replaced by canonical good value;
- struct list where old `.IP` is retained but another field changes;
- struct list where old `.IP` moves to a different index;
- new bad `.IP` in a struct list rejected.

## Phase 5: Field Classification

Before migrating a field, classify it by ratcheting support.

### Safe With Existing Declarative Ratcheting

These can be migrated after Phases 1-3:

- scalar IP fields;
- pointer IP fields;
- map values;
- map keys;
- `listType=set` string lists;
- `listType=map` lists where the IP field is the list-map key.

Examples:

- `Pod.Status.PodIP`;
- `Pod.Status.HostIP`;
- `Pod.Status.PodIPs[*].IP`, because `PodIPs` is `listType=map` keyed by `ip`;
- `EndpointSlice.Endpoints[*].Addresses`, because `Addresses` is `listType=set`.

### Requires Phase 4 Primitive

Do not migrate these with plain `+k8s:eachVal=+k8s:format=k8s-ip-sloppy`:

- `Service.Spec.ClusterIPs`, `listType=atomic`;
- `Service.Spec.ExternalIPs`, `listType=atomic`;
- `Pod.Status.HostIPs[*].IP`, parent list is `listType=atomic`;
- `Service.Status.LoadBalancer.Ingress[*].IP`, parent list is `listType=atomic`;
- `networking.Ingress.Status.LoadBalancer.Ingress[*].IP`, parent list is `listType=atomic`;
- `Endpoints.Subsets[*].Addresses[*].IP` and `NotReadyAddresses[*].IP`, parent lists are atomic and current handwritten logic has coarse subset ratcheting;
- `Service.Spec.LoadBalancerSourceRanges`, CIDR analogue, `listType=atomic`.

## Phase 6: Proof Migrations

Start with low-risk fields.

Recommended first proof PR:

1. Runtime validators.
2. Generator wiring.
3. Option plumbing for one strategy.
4. One safe field migration.

Good proof candidates:

- a scalar legacy IP field with `validOldIPs == nil`;
- `EndpointSlice.Endpoints[*].Addresses` with address-family checks left handwritten;
- `Pod.Status.PodIPs[*].IP` with handwritten validation retained initially for mismatch comparison.

For each proof migration:

- Add the declarative tag.
- Keep handwritten validation initially.
- Mark handwritten errors as covered only when mismatch tests prove equivalence.
- Test create and update with `StrictIPCIDRValidation` enabled and disabled.
- Test old bad values on update.

## Phase 7: Atomic-List Migrations

After Phase 4 lands, migrate atomic-list fields one by one.

For scalar slices:

- use the old-membership helper with identity projection;
- examples: `ClusterIPs`, `ExternalIPs`.

For struct slices:

- use the old-membership helper with `.IP` projection;
- examples: service load-balancer ingress and networking ingress load-balancer ingress.

Keep surrounding handwritten semantic checks until they have declarative equivalents.

Examples of checks not covered by IP format:

- IPv4-only / IPv6-only family checks;
- `ValidateEndpointIP` special-address rejection;
- `ClusterIP` / `ClusterIPs` `"None"` handling;
- dual-stack constraints;
- duplicate checks;
- immutability and repair rules;
- field-specific warning behavior.

## Phase 8: Warnings

Do not remove warning behavior as part of the initial format migration.

`GetWarningsForIP` and `GetWarningsForCIDR` are currently handwritten warning paths. Declarative validation does not currently provide an equivalent warning channel.

Track warnings as follow-up work:

- keep warnings handwritten;
- or add a declarative warnings mechanism;
- or explicitly scope warnings out of the first migration.

## Phase 9: CIDR Follow-Up

After IP is proven, repeat the same pattern for CIDR:

- `+k8s:format=k8s-cidr`;
- `+k8s:format=k8s-cidr-sloppy`;
- possibly `+k8s:format=k8s-interface-address`.

CIDR has the same feature gate and the same atomic-list ratcheting concerns, especially `LoadBalancerSourceRanges`.

## Suggested PR Sequence

1. Add `IP` and `IPSloppy` runtime validators plus unit tests.
2. Wire `k8s-ip` and `k8s-ip-sloppy` in validation-gen plus output tests.
3. Add `StrictIPCIDRValidation` option plumbing for one proof strategy.
4. Migrate one safe field while retaining handwritten validation for mismatch checking.
5. Add projected old-membership ratcheting helper and tests.
6. Migrate atomic-list IP fields one by one.
7. Add CIDR strict/sloppy formats.
8. Address warnings.
9. Remove handwritten validation only after declarative parity is proven.

## Acceptance Criteria

- `+k8s:format=k8s-ip` generates strict canonical IP validation.
- `+k8s:format=k8s-ip-sloppy` generates legacy gate-aware IP validation.
- `IPSloppy` matches `IsValidIPForLegacyField` for scalar and correlated old values.
- `StrictIPCIDRValidation` is supplied through `operation.Operation.Options`, not direct feature-gate imports in apimachinery.
- Atomic-list IP fields are not migrated until projected old-membership ratcheting exists.
- Existing bad stored IPs do not block unrelated updates.
- Field-specific semantics remain intact.
- Warnings are not accidentally removed.
- Handwritten/declarative mismatch tests pass before handwritten validation is removed.
