# sqlc Migration Plan

Migrate raw SQL queries in `queue/pg_queue.go` to type-safe sqlc-generated code. `Songs()` is already done as a proof of concept.

Approach: SELECTs first (read-only, no side effects), then INSERTs/UPDATEs/DELETEs, then complex multi-step operations last.

---

## Phase 1: Simple SELECTs

Straightforward reads with no dynamic SQL.

| Method | Query file |
|--------|-----------|
| `SignupsOpen()` | `db/queries/settings.sql` |
| `HasName()` | `db/queries/signups.sql` |
| `Performed()` | `db/queries/performed_songs.sql` |

These have fixed columns, no string-concatenated WHERE clauses, and no joins that require custom scan logic. Good candidates to establish the pattern.

---

## Phase 2: JOIN SELECTs

More complex reads that return multiple joined rows.

| Method | Notes |
|--------|-------|
| `Entries()` | Multi-table join; result requires `scanEntries` grouping logic — sqlc generates row structs, but the grouping (entries → songs) stays in Go |
| `AuthenticateHost()` | Simple scan but iterates rows for bcrypt comparison |
| `MoveEntry()` position read | The initial `SELECT id, position` inside `MoveEntry` |

---

## Phase 3: Simple INSERTs/UPDATEs/DELETEs

Single-statement writes with fixed parameters.

| Method | Notes |
|--------|-------|
| `MoveCurrentToBottom()` | UPDATE with subquery — uses `firstTodayID` and `todayQueueEntries` constants |
| `RemoveCurrent()` | DELETE with subquery |
| `AddSongToFirst()` existence check | `SELECT COUNT(*)` before the INSERT |
| `MoveEntry()` UPDATE | The final position write |
| `ToggleSignups()` UPDATE | RETURNING clause — sqlc handles this |
| `ToggleSignups()` DELETE | Simple date-scoped delete |
| `CompleteCurrentSong()` individual statements | Three statements inside a transaction |

---

## Phase 4: Dynamic / Multi-Step Operations

These require more care or may not be full sqlc candidates.

| Method | Notes |
|--------|-------|
| `Add()` song ID validation | Dynamic `IN (...)` clause with variable-length args — sqlc doesn't support dynamic IN; keep as raw SQL or restructure |
| `Add()` INSERT into signups | Position subquery — straightforward once the IN check is handled separately |
| `Add()` INSERT into entry_songs | Loop with per-song inserts — stays in Go loop |
| `CompleteCurrentSong()` transaction | Wraps three statements; transaction management stays in Go |

---

## Notes

- Run `sqlc generate` after each query file change and commit the generated `db/sqlcdb/` files
- Keep `db/schema.sql` in sync with any new Liquibase migrations before regenerating
- The `firstTodayID` and `todayQueueEntries` SQL constants currently embedded as Go strings will need to be inlined into the sqlc query files