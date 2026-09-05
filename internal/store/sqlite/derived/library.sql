-- library — the whole of the derived half, at generation 1.
--
-- This file is **not** a migration and this directory is **not** a lineage.
-- ADR-0003 says the derived half is dropped and rescanned on a schema change
-- and that no migration is ever written for it, so what is here is the whole
-- current schema rather than a step towards it: edit it in place, bump
-- `derivedGeneration`, and every installation rebuilds at its next start
-- (003 plan §6.8). `derivedSchemaDigest` is what stops the second half of that
-- sentence from being a thing somebody remembers to do.
--
-- Nothing here may reference a table in the precious half. Foreign keys are on
-- (ADR-0003, the writer DSN), and architecture §6's rule that no reference
-- points from the precious half into the derived one has its mirror image
-- here: a derived table naming `libraries(id)` would tie the half that is
-- rebuilt to the half that is kept, and the identifier string is what carries
-- that relation instead — which is exactly what makes 007's favourites and
-- 009's playlists survive a rescan.

-- items — one row per item of every type, including the container rows that
-- back no file at all.
CREATE TABLE items (
    -- 32 lowercase hex, **derived** from the library and the item's path
    -- relative to its root (003 §3.6, plan §6.3). Derived and not allocated is
    -- the whole reason this table can be dropped: a rebuild computes the same
    -- identifiers again, so every favourite, resume position and playlist row
    -- in the precious half still names the item it named before.
    id                  TEXT    PRIMARY KEY,

    -- The library's allocated identity, **as a string and not a foreign key**.
    -- See the header: the constraint would be real, foreign keys are on, and
    -- the rebuild this file exists for would then be a drop the engine has an
    -- opinion about.
    library_id          TEXT    NOT NULL,

    -- The container's id. NULL for a library's own root row, which is the one
    -- item in a library that hangs from nothing.
    parent_id           TEXT,

    -- One of the eight `library.Kind` values, spelled as the domain spells
    -- them. A second spelling would be a second identifier for every item of
    -- that type, because the type is an input to the derivation.
    type                TEXT    NOT NULL,

    -- What the path or the tags said. 004 may replace it.
    name                TEXT    NOT NULL,

    -- A query-pattern column in the strongest sense: it exists so that no
    -- ORDER BY ever calls a function or a collation. It is computed at write
    -- time (003 §3.7, plan §6.6) and compared with the default BINARY
    -- collation — NOCASE here would reorder half a library and is the
    -- collation mistake ADR-0003 names by name.
    sort_key            TEXT    NOT NULL,

    -- Relative to the root, exactly as the walk read it, and NULL for an
    -- inferred container that has no directory of its own. It is deliberately
    -- not the normalised key the identifier was derived from: that key is
    -- case-folded in a case-insensitive library and a path stored lower-cased
    -- cannot be opened on a case-sensitive filesystem at all.
    path                TEXT,

    -- Which of the library's roots `path` is relative to.
    root_ordinal        INTEGER,

    -- Episode, season, track and disc numbers, and the second number of a
    -- multi-episode file. Nullable because absent and zero are different
    -- answers: season 0 is `Specials` (003 §3.4) and a season with no number
    -- at all is not it.
    index_number        INTEGER,
    parent_index_number INTEGER,
    index_number_end    INTEGER,

    -- The year 003 §3.3 strips out of a name.
    production_year     INTEGER,

    -- Ticks — 100-nanosecond intervals since 0001-01-01T00:00:00Z, the unit
    -- the wire carries (behaviours §1.3) and therefore the unit storage
    -- carries. For the date-named episode of 003 §3.4.
    premiere_date       INTEGER,

    -- Whether the name said too little to place the item. Counted apart from a
    -- skip because 003 §3.8 requires the two be reported apart: an operator
    -- told a file was skipped goes looking for something that is not missing.
    --
    -- No DEFAULT, for `case_sensitive`'s reason one migration over: a default
    -- is a second place the value is decided, and every writer of this table
    -- has already decided it.
    unplaceable         INTEGER NOT NULL
) STRICT;

-- item_files — one row per file behind an item.
--
-- A table of its own rather than columns on `items`, because a multi-part film
-- is **one** item with two sources in order (003 §3.3) and 008 will read
-- exactly this table to answer `MediaSources`. A path column on `items` with a
-- second one beside it for part two is the shape that makes the third part a
-- migration — in the one half that is not allowed to have any.
CREATE TABLE item_files (
    -- Within the derived half, so the constraint is real and the cascade is
    -- what `RemoveItems` relies on rather than a second DELETE it could
    -- forget.
    item_id     TEXT    NOT NULL REFERENCES items(id) ON DELETE CASCADE,

    -- Part order for a multi-part film, 0 for everything else.
    ordinal     INTEGER NOT NULL,

    -- Relative to the root.
    path        TEXT    NOT NULL,

    -- Bytes. **Observable**, because a media source carries `Size`
    -- (behaviours §2.17).
    size        INTEGER NOT NULL,

    -- Ticks. Observable nowhere, and stored only as half of 003 plan §6.4's
    -- change signal.
    modified_at INTEGER NOT NULL,

    PRIMARY KEY (item_id, ordinal)
) STRICT;

-- scan_state — one row per library: the claim, and the last summary.
--
-- Derived, and it is the row in this file that was decided rather than
-- obvious. "When this library was last scanned" reads like operator-facing
-- history, and history is precious — but a library that has never been scanned
-- and a library whose derived half was just dropped are **the same state**, and
-- they have to be, because a generation bump drops this table and scans every
-- library from nothing. In the precious half it would leave a row saying
-- "scanned yesterday" over an empty `items`, and every reader of the pair would
-- have to decide which half to believe (003 plan §4.3).
--
-- A library with no row here has never been scanned. The row appears when a
-- scan claims the library and is not created before that.
CREATE TABLE scan_state (
    library_id       TEXT PRIMARY KEY,

    -- Which process is scanning, and since when. A claim older than
    -- `staleAfter` is broken and taken, because a process killed mid-scan
    -- leaves one behind and the alternative is a library nothing will ever
    -- scan again (003 plan §6.9). Both NULL when nothing holds the claim.
    claimed_at       INTEGER,
    claimed_by       TEXT,

    -- When the last scan finished, and whether it was a full re-examination.
    last_scan_at     INTEGER,
    last_scan_full   INTEGER,

    -- 003 §3.8's counts, as a document. A document rather than a column per
    -- count because nothing queries them: they are read back whole and shown
    -- to an operator.
    summary_document TEXT
) STRICT;
