-- 0003_libraries — the two tables 003 configures a library with.
--
-- Both are precious in ADR-0003's sense, and the filing is the whole of what
-- this migration decides. A library's identity is **allocated** and kept
-- (003 §3.6), so that renaming a library or moving its roots costs nothing —
-- and an allocated identifier is by definition not reconstructible from the
-- files, so these rows cannot live in a half that is dropped and rebuilt.
--
-- Two columns beside it are worse than the identifier if they are lost:
-- case_sensitive decides every identifier under the library, and
-- collection_type decides which resolution rules apply and therefore every
-- item's type as well as its identifier (003 plan §4.1). Filed under the
-- derived directory this file would create exactly the same tables and every
-- test of the SQL would pass over it; what it would have changed is that a
-- library scan would silently re-derive every identifier in the installation.
-- That is why the test asserts the version this migration moves and not only
-- the tables it creates.

-- libraries — one row per configured library.
CREATE TABLE libraries (
    -- 32 lowercase hex (behaviours §1.4's shape), and the one identifier in
    -- this feature that is **allocated** rather than derived. Deleting a
    -- library and declaring another with the same name and the same roots is
    -- not the same library, and every item under it gets a new identifier
    -- (003 §3.6).
    id              TEXT    PRIMARY KEY,

    -- As the operator spelled it. This is the name of the library's own
    -- CollectionFolder item, and it is editable: `atrium library rename`.
    name            TEXT    NOT NULL,

    -- A query-pattern column, and the same shape 002's username_folded is: the
    -- subcommand addresses a library by name because an operator has the name
    -- and never the allocated identifier. Stored rather than folded per query
    -- so that the uniqueness the subcommand depends on is the database's rule
    -- and not a convention — without it two libraries differing only in case
    -- are creatable, and the lookup that finds both has no defined answer.
    --
    -- The fold is the domain's and is applied before the value arrives here.
    -- SQLite compares TEXT by bytes, so what this refuses is exactly the pairs
    -- the caller folded together and no others.
    name_folded     TEXT    NOT NULL UNIQUE,

    -- movies, tvshows or music, spelled as `library.CollectionType` spells
    -- them. **Frozen after creation** (003 §3.6): it selects which resolution
    -- rules apply, so changing it re-resolves every file under a different set
    -- of rules and gives every item a new type and a new identifier.
    --
    -- The CHECK is a backstop rather than the refusal an operator meets. The
    -- subcommand refuses a fourth value at flag parsing and lists the three
    -- (003 plan §7); this is what stops a value that got past it from becoming
    -- a library whose type no resolver knows.
    collection_type TEXT    NOT NULL CHECK (collection_type IN ('movies', 'tvshows', 'music')),

    -- Whether paths compare with regard to case when an identifier is derived.
    -- **Frozen after creation** (003 §3.6), and the reason it is frozen is
    -- sharper than the type's: changing it rewrites every identifier under the
    -- library and nothing stores the old ones to undo with.
    --
    -- There is deliberately no DEFAULT. A default here would be a second place
    -- the value is decided, and 003 §3.6 makes it a property an operator
    -- states when the library is declared.
    case_sensitive  INTEGER NOT NULL,

    -- Ticks — 100-nanosecond intervals since 0001-01-01T00:00:00Z, which is
    -- the unit the wire carries (behaviours §1.3) and therefore the unit
    -- storage carries, so that no conversion can be forgotten at a boundary.
    created_at      INTEGER NOT NULL
) STRICT;

-- library_roots — one row per configured root.
--
-- A table of its own rather than a column holding a list, because the ordinal
-- is what `ScannedItem.RootOrdinal` indexes: an item's path is relative to one
-- of these, and a list encoded into a single column would make "which root is
-- root 1" a decoding rule rather than a key.
CREATE TABLE library_roots (
    -- ON DELETE CASCADE, which 003 plan §4.1's table did not spell and which
    -- writing `RemoveLibrary` forced: foreign keys are on (ADR-0003, the
    -- writer DSN), so a delete of a library holding roots is refused unless
    -- something removes them first. Making that the database's rule rather
    -- than the method's discipline is the same argument name_folded's
    -- uniqueness above is made with, and it is the one this project takes
    -- every time the choice comes up.
    library_id TEXT    NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,

    -- The order the operator gave them. It decides nothing — an item's
    -- identity comes from its path relative to its root and not from which
    -- root that is — but it is stored rather than left to a query's default
    -- ordering, because a list that moved between two reads would move every
    -- recorded RootOrdinal with it, and an item whose ordinal no longer names
    -- the root its path is relative to cannot be opened (architecture §2
    -- forbids an order derived from anything but stable input).
    ordinal    INTEGER NOT NULL,

    -- Absolute, as configured.
    path       TEXT    NOT NULL,

    PRIMARY KEY (library_id, ordinal)
) STRICT;
