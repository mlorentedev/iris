-- SPDX-License-Identifier: Apache-2.0
-- Copyright 2026 Manu Lorente
--
-- Down migration for 0001_init. Mirrors the up file in reverse. Migrations are
-- APPEND-ONLY once merged (bootstrap-contract section 3): never edit a merged up
-- file, always add a new numbered pair. This down exists for local rollback
-- during development, not for production schema surgery.
--
-- Dropping the table drops its trigger with it.

DROP TABLE IF EXISTS teams;
