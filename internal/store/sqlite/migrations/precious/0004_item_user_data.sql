-- 0004_item_user_data — the precious row 003 §3.8 requires outlive an item.
--
-- **Why this table is filed by 003 and not by 007, which owns the data.**
-- 003 §3.8 says *"user data outlives items"* and AC-11 is its criterion: a row
-- keyed on an item's identifier, written before the file is deleted, is still
-- there after the scan that removes the item, and names the item again when
-- the file comes back. 003 plan §6.5 explains why that costs the scanner
-- nothing — no precious row references the derived half by row id, and the
-- identifier is a function of the path — and then names the risk it leaves:
-- *"a later feature that 'tidies up' user data whose item is gone would break
-- AC-11 and nothing in this feature would fail."*
--
-- That risk is only assertable against a real row in a real precious table.
-- Without one AC-11's middle clause has no test until 007 lands, which is the
-- shape both of this project's closing audits caught — a criterion proven a
-- level too low, or not at all. So the table lands here, at the feature whose
-- criterion needs it, and 007 extends it.
--
-- **What it holds is the two nouns of 003 §3.8's own sentence** — *"must not
-- cost the user their favourites and resume position"* — and nothing else.
-- 007 §4 owns four more properties (played, play count, last played, and the
-- live playstate of a session); none of them is named by 003 and none of them
-- is here. A table shaped from 007's specification would be this feature
-- deciding another feature's storage, which is the opposite mistake.
--
-- **007 extends this table rather than replacing it.** The columns it adds are
-- a further precious migration; the two here keep their meaning. What must not
-- happen is a parallel table, because the assertion that guards all of it —
-- `internal/app`'s AC-11 test — is written against this one, and a sweep over
-- a table nothing watches is the failure plan §6.5 predicts arriving unseen.
CREATE TABLE item_user_data (
    -- The account the state belongs to. A real foreign key, because both
    -- tables are precious: architecture §6 forbids a reference from the
    -- precious half into the derived one, and this is a reference within one
    -- half, which is the ordinary case.
    user_id                 TEXT    NOT NULL REFERENCES users(id),

    -- The item's **derived** identifier, as a string and deliberately not a
    -- foreign key into `items`. This is the whole mechanism 003 plan §6.5
    -- describes: a constraint here would make the derived half's drop
    -- (plan §6.8) refuse, and `ON DELETE CASCADE` would make a scan that
    -- removed an item delete the user's favourite with it — which is exactly
    -- what §3.8 forbids. The row is allowed to name an identifier no `items`
    -- row currently has; that state is a file that is temporarily gone, not an
    -- orphan.
    item_id                 TEXT    NOT NULL,

    -- 003 §3.8's *"favourites"*.
    is_favourite            INTEGER NOT NULL,

    -- 003 §3.8's *"resume position"*, in ticks — 100-nanosecond intervals,
    -- which is the unit the wire carries (behaviours §1.3) and therefore the
    -- unit storage carries, so that no conversion can be forgotten at a
    -- boundary.
    playback_position_ticks INTEGER NOT NULL,

    PRIMARY KEY (user_id, item_id)
) STRICT;
