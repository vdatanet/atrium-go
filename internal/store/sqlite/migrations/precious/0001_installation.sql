-- 0001_installation — the single row this installation is.
--
-- spec 4 owns three pieces of observable state. The identity is not one of the
-- columns here: it lives in a file beside this database, because AC-4 asks that
-- it survive "a rebuild of the store from empty" and a row cannot (plan 4).
-- What is left is the friendly name and whether setup has finished.
--
-- The table is precious in ADR-0003's sense: nothing rebuilds it, so every
-- later change to it is a forward-only migration rather than a drop.

CREATE TABLE installation (
    -- One row, and the CHECK is what says so. A configuration table with two
    -- rows is a bug that reads as a mystery: every query returns whichever the
    -- planner reached first, and the answer changes with the schema (plan 4).
    id                 INTEGER PRIMARY KEY CHECK (id = 1),

    -- Operator-chosen friendly name, reported as ServerName (spec 3.1).
    server_name        TEXT    NOT NULL,

    -- The instant initial configuration finished, in ticks — a count of
    -- 100-nanosecond intervals since 0001-01-01T00:00:00Z, which is .NET's
    -- DateTime.Ticks and therefore the unit the wire uses (ADR-0003,
    -- behaviours 1.3). The origin is part of the unit and was left unsaid here
    -- until T4 had a type for it; plan 4 records it. NULL until setup is done:
    -- StartupWizardCompleted is this column being non-NULL, which is why the
    -- observable is a boolean and the column is not.
    setup_completed_at INTEGER
) STRICT;

-- A fresh installation is named "atrium" (spec 3.1). The row is seeded here
-- rather than written on a first start, so that every read of this table finds
-- a row and no caller has to carry a "not configured yet" branch.
INSERT INTO installation (id, server_name, setup_completed_at) VALUES (1, 'atrium', NULL);
