# Public Issues/PRs on managedFields Overhead

## 1) Issue #90066 (2020): managedFields verbosity pain
- URL: https://github.com/kubernetes/kubernetes/issues/90066
- Signal: users reported managedFields dominating `kubectl -o yaml` output.
- Insight: metadata size inflation was visible immediately to users.

## 2) PR #96878 (2020): strip managedFields in `kubectl get`
- URL: https://github.com/kubernetes/kubernetes/pull/96878
- Signal: CLI introduced controls to reduce managedFields noise in output.
- Insight: ecosystem already uses selective omission to improve usability and practical overhead.

## 3) PR #94986 (2020): omit managedFields in audit entries
- URL: https://github.com/kubernetes/kubernetes/pull/94986
- Signal: audit policy added `omitManagedFields` support.
- Insight: upstream accepted targeted omission as a valid tradeoff for log volume and storage pressure.

## 4) Issue #131175 (2025): no-op SSA updates timestamp/resourceVersion
- URL: https://github.com/kubernetes/kubernetes/issues/131175
- Signal: no-op apply behavior may still update metadata timestamps.
- Insight: metadata churn can create write amplification and cache churn even when business state is unchanged.

## 5) PR #131016 (2025): scheduler trimming bug fix
- URL: https://github.com/kubernetes/kubernetes/pull/131016
- Signal: scheduler code path explicitly trims managedFields for memory reasons.
- Insight: in-tree components already treat managedFields as costly for hot caches.

## 6) PR #136760 (open, 2026): omit managedFields in get/list option
- URL: https://github.com/kubernetes/kubernetes/pull/136760
- Signal: ongoing upstream exploration for server-side omission in read APIs.
- Insight: this is a concrete direction that aligns with large-cluster memory and payload concerns.

## Synthesis
- There is no single merged “managedFields memory fix”.
- Instead, public history shows repeated targeted mitigations:
  - hide/omit in specific views,
  - avoid logging it where not needed,
  - trim in memory-sensitive controllers,
  - explore omission options in read APIs.
- This supports a layered mitigation strategy rather than a one-shot redesign.
