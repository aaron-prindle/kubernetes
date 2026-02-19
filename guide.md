# The ManagedFields Memory Bottleneck: Meeting Guide & Technical Strategy

This guide synthesizes the recent conversations, GitHub issues, and architectural pivots regarding the `managedFields` memory bottleneck. It is designed to prepare you for your meeting with Joe Betz and Marek Siarkowicz.

## 1. The "Why": The Customer and the Urgency
*   **The Customer:** "Pine" (internal Google code name for Anthropic).
*   **The Scale:** They are running massive "megaclusters" (up to 65k nodes).
*   **The Problem:** The Kubernetes control plane (KCP) is running out of memory. `kube-apiserver` and custom controllers are consuming massive amounts of RAM, requiring huge VMs (C4 family, which are in short supply).
*   **The Root Cause:** Server-Side Apply (SSA) attaches `managedFields` to every object. This metadata accounts for **>50% of the stored resource size**. In highly replicated workloads (e.g., a DaemonSet with 50k pods), this same structural metadata is duplicated 50,000 times in memory.
*   **The Constraint:** Joe Betz emphasized this needs to be delivered **ASAP**. The solution must be **simple** and **rock-solid** (low risk of causing crashes or data corruption).

---

## 2. The Evolution of the Solution (Clearing up the confusion)

You were originally working on a plan to deduplicate `[]byte` slices inside the apiserver's watch cache. **You need to put that specific plan on hold.** The strategy has pivoted. Here is exactly what happened:

1.  **Marek's Experiments:** Marek tested several interning (deduplication) strategies. He found that interning `ManagedFields` strings could save ~25% of memory across the *entire* control plane (not just the watch cache). However, when he tried to deduplicate whole `PodSpecs`, the apiserver **crashed** (`fatal error: concurrent map iteration and map write`). This proved that sharing mutable memory across Kubernetes objects is extremely dangerous because hidden mutations happen during the object lifecycle.
2.  **Jordan Liggitt's Pivot:** Jordan looked at the plan to deduplicate `FieldsV1.Raw` (which is a `[]byte`). He pointed out that `[]byte` is mutable. If we share one `[]byte` across 50,000 Pods, and one component accidentally modifies it, it corrupts all 50,000 Pods and crashes the apiserver.
3.  **The New Alignment:** To fix the safety issue, Jordan proposed a fundamental API type change: **Change `metav1.FieldsV1.Raw` from a `[]byte` to a `string`**.
    *   In Go, strings are immutable.
    *   If it's a string, it is 100% safe to share it across thousands of objects. No concurrent mutation panics are possible.
4.  **Joe's Blessing:** Joe agreed completely ("+100") and officially tagged you on GitHub to execute this new path.

---

## 3. The Aligned Solution (What you are building)

The solution you need to build is actually simpler than your original watch cache plan, but it touches lower-level core API machinery. It consists of two parts:

### Part A: The Type Refactor (Jordan's PoC)
You need to change the Go struct definition of `FieldsV1`.
*   **From:** `Raw []byte`
*   **To:** `Raw string`
*   **The Catch:** You must ensure that JSON, Protobuf, and CBOR serialization/deserialization behaves *exactly* the same on the wire. The API over the network cannot change; only the in-memory Go representation changes. Jordan's PoC (`commit 2e77c9a...`) provides the blueprint for this.

### Part B: The Actual Interning
Once `FieldsV1.Raw` is a string, you can deduplicate it as it is deserialized from etcd or the network.
*   Because Go 1.23 introduced the `unique` package, you can likely just wrap the deserialized string in `unique.Make(string).Value()`.
*   This will automatically point all identical `managedFields` strings to the exact same memory address, saving gigabytes of RAM.

---

## 4. Why this fits the "ASAP / Simple & Solid" Mandate
*   **Simple:** We don't need to build a complex, locked caching pool in `watch_cache.go`. We just use Go's native string properties and the `unique` package at the deserialization boundary.
*   **Solid (Safe):** Because strings are immutable, we completely eliminate the risk of race conditions, memory corruption, and the concurrent map crashes Marek experienced. It is mathematically safe to share.

---

## 5. Meeting Prep: What to discuss with Joe and Marek

When you meet with them, you should project confidence that you understand the pivot to `string` and are ready to execute it. Here is your agenda/questions for the meeting:

1.  **Confirm the Pivot:** "Just to ensure we are 100% aligned: I am putting the `[]byte` watch-cache intern pool on hold. My primary focus is taking Jordan's `FieldsV1` string PoC to the finish line. Is that correct?"
2.  **Serialization Risks:** "Changing an API type from `[]byte` to `string` is safe in memory, but we need to ensure 100% wire compatibility for Protobuf, JSON, and CBOR. Are there any specific edge cases in the Protobuf generation (`generated.pb.go`) that Jordan's PoC might have missed?"
3.  **The Interning Mechanism:** "Once it's a string, how do we want to intern it? Should I implement this using Go 1.23's `unique` package directly in the protobuf/JSON decoders, or do we want a custom string intern pool?"
4.  **Testing Strategy:** "To ensure this is 'rock-solid' for Pine, my plan is to rely heavily on existing serialization fuzzers and add specific tests to prove identical memory addresses are returned for identical `managedFields`. Does that sound sufficient for the testing bar?"
5.  **Rollout / Feature Gates:** "Because this touches core deserialization, do we need to hide the *interning* part behind a feature gate, or do we just roll it out unconditionally since string interning is theoretically invisible to the rest of the system?"
