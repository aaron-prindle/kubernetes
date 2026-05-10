# Declarative IP Validation PR Notes

## Summary

This PR introduces declarative IP format validators and attempts a first proof migration for `PodStatus.PodIPs[*].IP`.

The migration is intentionally narrow:

- `+k8s:format=k8s-ip` maps to strict canonical IP validation.
- `+k8s:format=k8s-ip-sloppy` maps to legacy, feature-gated IP validation.
- `PodStatus.PodIPs[*].IP` uses `k8s-ip-sloppy` because handwritten validation uses `IsValidIPForLegacyField`.
- `PodStatus.PodIPs` is declared as a declarative `listType=map` keyed by `ip`, so unchanged legacy IP strings can ratchet by key.

## Important Caveat: Pod Subresource Scope

Generated validation currently registers `Pod` for all declared Pod subresources:

```go
case "/", "/ephemeralcontainers", "/status":
```

This is mechanically correct for the `Pod` type because the type declares:

```go
// +k8s:supportsSubresource="/status"
// +k8s:supportsSubresource="/ephemeralcontainers"
```

However, the migrated handwritten IP validation for `status.podIPs[*].ip` only runs from the status update path:

```go
ValidatePodStatusUpdate(...)
  -> validatePodIPs(...)
```

That means the generated registration is broader than the handwritten callsite for this specific check.

In normal apiserver flow this should be inert outside `/status` because strategy preparation prevents users from mutating status on the other paths:

- root create: `podStrategy.PrepareForCreate` resets `pod.Status`;
- root update: `podStrategy.PrepareForUpdate` restores `newPod.Status = oldPod.Status`;
- ephemeralcontainers update: `dropNonEphemeralContainerUpdates` restores `newPod.Status = oldPod.Status`;
- status update: status is mutable and `validatePodIPs` runs today.

Still, this is not a perfect callsite match. A direct generated-validation invocation against a root Pod object with populated `status.podIPs` could validate a field that handwritten root Pod validation does not validate.

## PR Description Callout

Call this out explicitly in the final PR description:

> Note: validation-gen registers `Pod` validation for `/`, `/status`, and `/ephemeralcontainers` because those are the subresources declared by the type. The migrated `status.podIPs[*].ip` check corresponds to handwritten `/status` validation. In normal apiserver request flow, root and ephemeral updates restore or clear status before validation, so this should not change user-visible behavior. We added/should add equivalence coverage for these paths, and a future generator-level subresource-scoping mechanism may be needed if we want field validations to exactly mirror handwritten callsites.

## Follow-Up Before Final PR

- Add/confirm tests showing root create/update and ephemeralcontainers update do not produce extra declarative `podIPs` IP errors.
- Decide whether to keep this as an acceptable proof migration caveat or introduce a field-level subresource scoping mechanism.
- Re-check empty `PodIP.IP` behavior because `+k8s:optional` short-circuits `k8s-ip-sloppy`.
- Re-check duplicate `PodIPs` behavior because declarative `listType=map` emits a uniqueness check.
