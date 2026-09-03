-- 0002_users_and_sessions — the four tables 002 owns.
--
-- All four are precious in ADR-0003's sense, and the filing is the point: a
-- rescan rebuilds a library, and it must not log anybody out (002 plan 4). The
-- two halves are two lineages carrying two versions, so a file put in the
-- derived directory would be a schema a rescan is entitled to drop — which is
-- an account and a credential deleted by a library scan. Nothing about the SQL
-- below would look different, which is why the test asserts the version this
-- migration moves and not only the tables it creates.
--
-- 002 plan 4 grades the four: users and user_credentials are precious in the
-- strong sense, reconstructible from nothing; sessions and access_tokens are
-- precious in the weak sense, not rebuildable by a rescan but reconstructible
-- by the user at the cost of one login. Nothing here acts on that distinction.
-- It is recorded there so that a repair one day can.

-- users — one row per account.
CREATE TABLE users (
    -- 32 lowercase hex (behaviours 1.4's shape), derived from the folded name
    -- rather than random, so an installation provisioned twice with the same
    -- names has the same identifiers (Principle VII, 002 plan 6.9).
    id                         TEXT    PRIMARY KEY,

    -- As the operator spelled it. This is what the user object's Name returns.
    username                   TEXT    NOT NULL,

    -- A query-pattern column, and the one constraint in this file that carries
    -- an argument of its own. Spec 3.3 matches a username case-insensitively,
    -- so this is the only column an authentication reads to find a row. It is
    -- stored rather than folded per query so that the uniqueness the login
    -- depends on is the database's rule and not a convention: without it, two
    -- accounts differing only in case are creatable, and the login that finds
    -- both has no defined answer — which is a credential check deciding
    -- between two credentials.
    username_folded            TEXT    NOT NULL UNIQUE,

    -- The serialised policy and configuration models (002 plan 4). Documents
    -- and not columns, because their declaration order is the wire order of an
    -- L3 body: adding a property is a code change, never a migration.
    policy_document            TEXT    NOT NULL,
    configuration_document     TEXT    NOT NULL,

    -- State, not policy, even though InvalidLoginAttemptCount is reported
    -- inside the policy object (spec 3.5). It moves on every failed login, and
    -- keeping it in the document would rewrite every permission on each
    -- failure. It is overlaid into the policy when the user object is built
    -- (002 plan 6.6).
    invalid_login_attempt_count INTEGER NOT NULL,

    -- Ticks — 100-nanosecond intervals since 0001-01-01T00:00:00Z, .NET's
    -- DateTime.Ticks, which is the unit the wire carries (behaviours 1.3).
    -- NULL, and only here: it is what makes LastLoginDate *absent* until the
    -- first login rather than reported as the minimum date (spec 3.5).
    last_login_at              INTEGER,
    last_activity_at           INTEGER
) STRICT;

-- user_credentials — zero or one row per account.
--
-- A table of its own rather than two columns on users, for the reason 002
-- plan 4 states: every read of a user object is a read of users, and none of
-- them wants the verifier in memory. The separation makes that a property of
-- the SQL rather than of everybody's discipline.
CREATE TABLE user_credentials (
    user_id    TEXT    PRIMARY KEY REFERENCES users(id),

    -- ADR-0006's PHC record, which carries its own parameters:
    -- $argon2id$v=19$m=65536,t=3,p=2$<salt>$<key>. A record derived below the
    -- current constants is what a rehash-on-next-login reads.
    phc        TEXT    NOT NULL,

    -- Ticks. What a rehash moves, and the only way to see one happened.
    written_at INTEGER NOT NULL
) STRICT;

-- sessions — one row per (client, device_id), with the user as a column.
CREATE TABLE sessions (
    -- 32 lowercase hex, derived from (client, device_id) — 002 plan 6.5.
    id                        TEXT    PRIMARY KEY,

    user_id                   TEXT    NOT NULL REFERENCES users(id),

    -- The four components a client identifies itself with (spec 3.2), echoed
    -- back by /Sessions.
    client                    TEXT    NOT NULL,
    device_id                 TEXT    NOT NULL,
    device_name               TEXT    NOT NULL,
    application_version       TEXT    NOT NULL,

    remote_endpoint           TEXT    NOT NULL,

    -- The declaration POST /Sessions/Capabilities/Full posted, whole. NULL
    -- until one is posted, which is an absence and not an empty declaration:
    -- storing the raw document is what makes behaviours 5.9's divergence the
    -- stated one rather than an accident (002 plan 6.10).
    capabilities_document     TEXT,

    created_at                INTEGER NOT NULL,
    last_activity_at          INTEGER NOT NULL,

    -- Ticks, and NOT NULL deliberately: the zero tick is a value here, not a
    -- missing one. Spec 3.3 measures LastPlaybackCheckIn as
    -- 0001-01-01T00:00:00.0000000Z for a session that has never played
    -- anything — .NET's minimum date, "not null and not absent"
    -- [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
    -- A nullable column would answer an absence, and the only honest thing to
    -- write for it on the wire would be the same date this column can already
    -- hold.
    last_playback_check_in_at INTEGER NOT NULL,

    -- The key 002 plan 4 states in prose, made the database's rule. It is
    -- redundant against the primary key today, because id is derived from
    -- exactly this pair (002 plan 6.5) — and that redundancy is what it is
    -- for: it is the constraint that fails on the day the derivation stops
    -- deriving from these two columns, which nothing else in the schema would
    -- notice.
    UNIQUE (client, device_id)
) STRICT;

-- access_tokens — one row per live token.
--
-- The token is stored as a digest and not as itself: ADR-0006's threat model
-- is the store file leaking, and a leaked table of live bearer tokens is that
-- leak with the hashing skipped (002 plan 4).
CREATE TABLE access_tokens (
    -- Unsalted SHA-256 of the token, lowercase hex. Unsalted deliberately: the
    -- input is 128 bits of uniform randomness, so a salt would defend against
    -- precomputation over a space nobody can precompute, at the cost of the
    -- primary-key lookup that makes a per-request check one indexed read.
    token_digest TEXT    PRIMARY KEY,

    user_id      TEXT    NOT NULL REFERENCES users(id),

    -- The foreign key that makes a token without a session impossible. A token
    -- naming a session that is not there would resolve to a caller with no
    -- client, no device and no activity to stamp, and every route that reads
    -- one would have to carry a branch for a state the schema can forbid.
    session_id   TEXT    NOT NULL REFERENCES sessions(id),

    -- A query-pattern column, duplicating the session's. 002 plan 6.5's
    -- replacement rule is keyed on (user, device) and the session's key is
    -- (client, device), so the two cannot be reached through one another.
    device_id    TEXT    NOT NULL,

    created_at   INTEGER NOT NULL
) STRICT;

-- The index the replacement rule reads. Revoking a user's tokens for one
-- device is the only query that goes to this table by anything but its primary
-- key, and without this it is a scan of every live token on the installation.
CREATE INDEX access_tokens_by_user_and_device ON access_tokens (user_id, device_id);
