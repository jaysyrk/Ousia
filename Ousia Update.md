# Lock-Free Atomic State Architecture Implementation Plan

This document outlines the step-by-step implementation of the lock-free state engine for Ousia, designed to handle 2,141+ RPS without Garbage Collection or Mutex bottlenecks.

## Phase 1: Memory Sharding (The 256-Bucket Router)
**Objective**: Eliminate the global `sync.Mutex` in `ratelimit.go` that blocks all threads.
**Actions**: 
- Create an array of 256 independent memory shards.
- Implement a fast hashing algorithm (like FNV-1a) to hash incoming IP addresses.
- Map each IP to one specific shard, evenly distributing the lock contention across sectors of the CPU cache.

## Phase 2: Hardware-Level Atomics (`sync/atomic`)
**Objective**: Remove the remaining locks inside the shards.
**Actions**:
- Replace traditional locking with `sync/atomic` operations (e.g., `atomic.AddInt64`, `atomic.CompareAndSwap`).
- Allow concurrent threads to fetch and update rate limit tokens at the hardware level in a single clock cycle without pausing.

## Phase 3: Object Recycling (`sync.Pool`)
**Objective**: Achieve zero-allocation during the networking hot-path to prevent GC pauses.
**Actions**:
- Set up a `sync.Pool` to hold pre-allocated request/metadata structs.
- Instead of creating a new object on every request, the engine will borrow an object, populate it, process the update, and return it.

## Phase 4: The Asynchronous Batch Flush
**Objective**: Isolate fast memory from slow I/O.
**Actions**:
- Implement a background ticker that wakes up every 1 second.
- Sweep the atomic counters, generate a snapshot, and push it to an async logging channel/database without blocking the ingress traffic.

---

### Previous Architecture Notes
(Kept for historical reference)
To keep Ousia running at native hardware speeds, the system replaces locks entirely by combining three core engineering principles: Hardware Atomics, Memory Sharding, and Object Recycling, culminating in an Asynchronous Batch Flush for safe storage.
