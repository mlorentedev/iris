---
id: lesson-002-imagepull-always-contacts-the-registry-gate-o
type: lesson
status: active
created: "2026-05-01"
owner: manu
tags: [iris, lesson]
---

# `ImagePull` always contacts the registry — gate on local presence

- **Context:** `docker.Runtime.Start` pulled the image before creating the container.
- **Problem:** `client.ImagePull` is **not** a local short-circuit — it always hits the registry to check the manifest. Pulling on every `Start` made the call network-bound and flaky (a stalled pull turned one conformance subtest into a 169s failure).
- **Solution:** `ensureImage` lists local images with a `reference=<ref>` filter and only calls `ImagePull` when none match. `Start` is now fast and offline-tolerant once an image is cached.
- **Tags:** docker, image-pull, performance, offline, flaky-tests
