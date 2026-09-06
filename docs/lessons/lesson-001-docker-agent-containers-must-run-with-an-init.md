---
id: lesson-001-docker-agent-containers-must-run-with-an-init
type: lesson
status: active
created: "2026-06-16"
owner: manu
tags: [iris, lesson]
---

# Docker agent containers must run with an init (PID 1 signal handling)

- **Context:** SDD-034c Docker substrate — each agent is one container; `Stop` calls `ContainerStop` (graceful SIGTERM, then SIGKILL after a timeout).
- **Problem:** the conformance suite's teardown took ~10s per agent. A process running as **PID 1** does not get the kernel's default signal dispositions, so `busybox sleep` (and any agent process) **ignores SIGTERM** — `ContainerStop` waited the full 10s timeout before SIGKILL on every Stop.
- **Solution:** create containers with `HostConfig.Init = true`. Docker injects a tiny init (tini) as PID 1 that forwards signals to the agent and reaps zombies. Teardown became instant and it is the correct posture for agents that spawn child processes anyway.
- **Tags:** docker, runtime, signals, pid1, graceful-shutdown
