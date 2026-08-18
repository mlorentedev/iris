---
id: lesson-003-moby-github-com-docker-docker-is-incompatible
type: lesson
status: active
created: "2026-05-01"
owner: manu
tags: [iris, lesson]
---

# moby `github.com/docker/docker` is `+incompatible` — pin `go-connections`

- **Context:** adding the Docker SDK (`github.com/docker/docker/client` at `v28.3.3+incompatible`).
- **Problem:** moby ships **no `go.mod`** (hence `+incompatible`), so `go mod tidy` has no constraint on its transitive deps and resolved `github.com/docker/go-connections@v0.7.0`. v0.7.0 **removed `sockets.DialPipe`**, which docker v28.3.3's `client` package still references → `undefined: sockets.DialPipe` build failure. Running `go get go-connections@latest` makes this worse, not better.
- **Solution:** pin `github.com/docker/go-connections v0.5.0` (the last line with `sockets.DialPipe`) in `go.mod`. When bumping the docker SDK, re-verify the compatible `go-connections` version rather than letting `@latest` win.
- **Tags:** go-modules, docker-sdk, plus-incompatible, dependency-pinning, build-break
