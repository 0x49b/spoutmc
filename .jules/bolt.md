## 2024-05-24 - [Avoid Redundant Docker Inspect Calls]
**Learning:** The Docker API `ContainerList` endpoint (which returns `[]container.Summary`) already includes the container's labels in the `Labels` field. There is no need to perform an expensive O(N) `ContainerInspect` round-trip for each container simply to check its labels.
**Action:** Always check the properties available on the `container.Summary` struct (like `Labels`, `Image`, `State`, `Status`, `NetworkSettings`) before reaching for `ContainerInspect` in loops over multiple containers.
