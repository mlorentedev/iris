// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package protocol

import (
	"errors"
	"fmt"
	"regexp"
)

// SubjectKind is a per-team channel on the bus. The subject hierarchy is
// team.<team>.<kind> (with an extra worker name for SubjectWorker).
type SubjectKind string

// The per-team subject channels.
const (
	SubjectLeader   SubjectKind = "leader"   // inbound to the team's leader agent
	SubjectActivity SubjectKind = "activity" // broadcast of activity events
	SubjectSystem   SubjectKind = "system"   // motor -> all components of the team
	SubjectWorker   SubjectKind = "worker"   // inbound to a specific worker (needs a name)
)

// nameRE is the single source of truth for valid team and worker names:
// lowercase alphanumerics and internal hyphens, no leading/trailing hyphen, no
// underscores, length >= 2. It also rejects the NATS-reserved characters
// '.', '*' and '>' (they are simply not in the allowed set), so a name can
// never break out of its subject token.
var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// Sentinel errors, comparable with errors.Is.
var (
	ErrInvalidName        = errors.New("protocol: invalid team/worker name")
	ErrInvalidSubjectKind = errors.New("protocol: invalid subject kind")
)

// ValidName reports whether s is a usable team or worker name. It is the SOLE
// validation chokepoint; both SubjectFor and any caller that needs to validate
// a name ahead of time go through it.
func ValidName(s string) bool {
	return nameRE.MatchString(s)
}

// SubjectFor builds the NATS subject for a team channel. It is the SOLE
// construction path for subjects (never string-concatenate at call sites), so
// name validation runs in exactly one place. For SubjectWorker, pass the worker
// name as the single variadic argument; the other kinds take none.
func SubjectFor(team string, kind SubjectKind, worker ...string) (string, error) {
	if !ValidName(team) {
		return "", fmt.Errorf("%w: team %q", ErrInvalidName, team)
	}

	switch kind {
	case SubjectLeader, SubjectActivity, SubjectSystem:
		if len(worker) != 0 {
			return "", fmt.Errorf("%w: %q takes no worker name", ErrInvalidSubjectKind, kind)
		}
		return fmt.Sprintf("team.%s.%s", team, kind), nil
	case SubjectWorker:
		if len(worker) != 1 {
			return "", fmt.Errorf("%w: worker kind needs exactly one worker name", ErrInvalidName)
		}
		if !ValidName(worker[0]) {
			return "", fmt.Errorf("%w: worker %q", ErrInvalidName, worker[0])
		}
		return fmt.Sprintf("team.%s.worker.%s", team, worker[0]), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidSubjectKind, kind)
	}
}
