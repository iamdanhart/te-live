package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/iamdanhart/te-live/db/sqlcdb"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidSongID = errors.New("one or more song IDs are invalid")

// todayFilter is appended to WHERE clauses on signups and performed_songs
// to scope all queries to the current calendar day.
const todayQueueEntries = `created_at >= CURRENT_DATE`

// firstTodayID is a subquery that returns the id of the first entry in today's queue.
const firstTodayID = `(SELECT id FROM telive.signups WHERE created_at >= CURRENT_DATE ORDER BY position ASC LIMIT 1)`

type PgQueue struct {
	db      *sql.DB
	queries *sqlcdb.Queries
}

func (q *PgQueue) Close() error {
	return q.db.Close()
}

func NewPgQueue(dsn string) (*PgQueue, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PgQueue{db: db, queries: sqlcdb.New(db)}, nil
}

func (q *PgQueue) Songs(ctx context.Context) []Song {
	rows, err := q.queries.ListSongs(ctx)
	if err != nil {
		slog.Error("Songs query", "err", err)
		return []Song{}
	}
	songs := make([]Song, len(rows))
	for i, r := range rows {
		songs[i] = Song{ID: int(r.ID), Title: r.Title, Artist: r.Artist, TabUrl: r.TabUrl}
	}
	return songs
}

func (q *PgQueue) Entries(ctx context.Context) []Entry {
	rows, err := q.db.QueryContext(ctx, `
		SELECT qe.id, qe.name, s.id, s.title, s.artist, s.tab_url, es.performed, qe.times_on_stage
		FROM telive.signups qe
		JOIN telive.entry_songs es ON es.entry_id = qe.id
		JOIN telive.songs s ON s.id = es.song_id
		WHERE `+todayQueueEntries+`
		ORDER BY qe.position ASC, es.sort_order ASC`)
	if err != nil {
		slog.Error("Entries query", "err", err)
		return nil
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (q *PgQueue) SignupsOpen(ctx context.Context) bool {
	value, err := q.queries.GetSignupsOpen(ctx)
	if err != nil {
		slog.Error("SignupsOpen query", "err", err)
		return false
	}
	return value == "true"
}

func (q *PgQueue) ToggleSignups(ctx context.Context) (bool, error) {
	var value string
	err := q.db.QueryRowContext(ctx, `
		UPDATE telive.settings
		SET value = CASE WHEN value = 'true' THEN 'false' ELSE 'true' END
		WHERE key = 'signups_open'
		RETURNING value`).Scan(&value)
	if err != nil {
		slog.Error("ToggleSignups query", "err", err)
		return false, err
	}
	if value == "true" {
		_, err = q.db.ExecContext(ctx, `DELETE FROM telive.signups WHERE created_at < CURRENT_DATE`)
		if err != nil {
			slog.Error("ToggleSignups clear old signups", "err", err)
		}
	}
	return value == "true", nil
}

func (q *PgQueue) Add(ctx context.Context, name string, songIDs []int) error {
	placeholders := make([]string, len(songIDs))
	args := make([]any, len(songIDs))
	for i, id := range songIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	var count int
	if err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM telive.songs WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	).Scan(&count); err != nil {
		return err
	}
	if count != len(songIDs) {
		return ErrInvalidSongID
	}

	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var entryID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO telive.signups (name, position)
		VALUES ($1, COALESCE((SELECT MAX(position) FROM telive.signups WHERE `+todayQueueEntries+`), 0) + 1)
		RETURNING id`, name).Scan(&entryID)
	if err != nil {
		return err
	}

	for i, songID := range songIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO telive.entry_songs (entry_id, song_id, sort_order)
			VALUES ($1, $2, $3)`,
			entryID, songID, i)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (q *PgQueue) MoveCurrentToBottom(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE telive.signups
		SET position = (SELECT MAX(position) FROM telive.signups WHERE `+todayQueueEntries+`) + 1
		WHERE id = `+firstTodayID)
	if err != nil {
		slog.Error("MoveCurrentToBottom", "err", err)
	}
	return err
}

func (q *PgQueue) RemoveCurrent(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM telive.signups WHERE id = `+firstTodayID)
	if err != nil {
		slog.Error("RemoveCurrent", "err", err)
	}
	return err
}

func (q *PgQueue) CompleteCurrentSong(ctx context.Context, singer string, songID int) error {
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("CompleteCurrentSong begin tx", "err", err)
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE telive.entry_songs SET performed = true
		WHERE entry_id = `+firstTodayID+`
		  AND song_id = $1`,
		songID)
	if err != nil {
		slog.Error("CompleteCurrentSong mark performed", "err", err)
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO telive.performed_songs (singer, song_id)
		VALUES ($1, $2)`,
		singer, songID)
	if err != nil {
		slog.Error("CompleteCurrentSong insert performed_songs", "err", err)
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE telive.signups SET times_on_stage = times_on_stage + 1
		WHERE id = `+firstTodayID)
	if err != nil {
		slog.Error("CompleteCurrentSong increment times_on_stage", "err", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("CompleteCurrentSong commit", "err", err)
		return err
	}
	return nil
}

func (q *PgQueue) Performed(ctx context.Context) []PerformedSong {
	rows, err := q.queries.ListPerformedToday(ctx)
	if err != nil {
		slog.Error("Performed query", "err", err)
		return nil
	}
	result := make([]PerformedSong, len(rows))
	for i, r := range rows {
		result[i] = PerformedSong{
			Singer: r.Singer,
			Song:   Song{ID: int(r.ID), Title: r.Title, Artist: r.Artist, TabUrl: r.TabUrl},
		}
	}
	return result
}

func (q *PgQueue) AddSongToFirst(ctx context.Context, songID int) error {
	var count int
	if err := q.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM telive.songs WHERE id = $1`, songID,
	).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return ErrInvalidSongID
	}
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO telive.entry_songs (entry_id, song_id, sort_order)
		VALUES (
			`+firstTodayID+`,
			$1,
			COALESCE((SELECT MAX(sort_order) FROM telive.entry_songs WHERE entry_id = `+firstTodayID+`), 0) + 1
		)`, songID)
	if err != nil {
		slog.Error("AddSongToFirst", "err", err)
	}
	return err
}

func (q *PgQueue) HasName(ctx context.Context, name string) bool {
	exists, err := q.queries.HasName(ctx, name)
	if err != nil {
		slog.Error("HasName query", "err", err)
		return false
	}
	return exists
}

type positionRow struct {
	id  int
	pos float64
}

// isNoOpMove reports whether id is already immediately after afterID in entries,
// meaning a move would produce the same order and should be skipped.
func isNoOpMove(entries []positionRow, id, afterID int) bool {
	for i, e := range entries {
		if e.id != id {
			continue
		}
		if afterID == 0 {
			return i == 0
		}
		return i > 0 && entries[i-1].id == afterID
	}
	return false
}

// computeNewPosition returns the position value that places the moved entry
// after the entry with afterID. afterID=0 means move to front; this relies on
// Postgres SERIAL IDs starting at 1, so 0 is never a valid entry ID.
// Returns false if afterID is non-zero and not found in entries.
func computeNewPosition(entries []positionRow, afterID int) (float64, bool) {
	if afterID == 0 {
		if len(entries) == 0 {
			return 1, true
		}
		return entries[0].pos - 1, true
	}
	for i, e := range entries {
		if e.id == afterID {
			if i == len(entries)-1 {
				return e.pos + 1, true
			}
			return (e.pos + entries[i+1].pos) / 2, true
		}
	}
	return 0, false
}

func (q *PgQueue) MoveEntry(ctx context.Context, id, afterID int) error {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, position FROM telive.signups
		WHERE `+todayQueueEntries+`
		ORDER BY position ASC`)
	if err != nil {
		slog.Error("MoveEntry query", "err", err)
		return err
	}
	defer rows.Close()

	var entries []positionRow
	for rows.Next() {
		var r positionRow
		if err := rows.Scan(&r.id, &r.pos); err != nil {
			slog.Error("MoveEntry scan", "err", err)
			return err
		}
		entries = append(entries, r)
	}
	if err := rows.Err(); err != nil {
		slog.Error("MoveEntry rows", "err", err)
		return err
	}

	if isNoOpMove(entries, id, afterID) {
		return nil
	}

	newPos, ok := computeNewPosition(entries, afterID)
	if !ok {
		slog.Warn("MoveEntry afterID not found", "afterID", afterID)
		return fmt.Errorf("afterID %d not found in today's queue", afterID)
	}

	result, err := q.db.ExecContext(ctx, `UPDATE telive.signups SET position = $1 WHERE id = $2`, newPos, id)
	if err != nil {
		slog.Error("MoveEntry update", "err", err)
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		slog.Error("MoveEntry rows affected", "err", err)
		return err
	}
	if n == 0 {
		slog.Warn("MoveEntry entry not found", "id", id)
		return fmt.Errorf("entry %d not found in today's queue", id)
	}
	return nil
}

// scanEntries collapses the joined rows into []Entry, grouping songs by singer.
func scanEntries(rows *sql.Rows) []Entry {
	var entries []Entry
	entryIndex := map[int]int{} // queue_entry id → index in entries slice

	for rows.Next() {
		var (
			entryID      int
			name         string
			songID       int
			title        string
			artist       string
			tabUrl       string
			performed    bool
			timesOnStage int
		)
		if err := rows.Scan(&entryID, &name, &songID, &title, &artist, &tabUrl, &performed, &timesOnStage); err != nil {
			slog.Error("scanEntries scan", "err", err)
			continue
		}
		idx, exists := entryIndex[entryID]
		if !exists {
			entries = append(entries, Entry{ID: entryID, Name: name, TimesOnStage: timesOnStage})
			idx = len(entries) - 1
			entryIndex[entryID] = idx
		}
		entries[idx].Songs = append(entries[idx].Songs, SongEntry{
			Song:      Song{ID: songID, Title: title, Artist: artist, TabUrl: tabUrl},
			Performed: performed,
		})
	}
	return entries
}

func (q *PgQueue) AuthenticateHost(ctx context.Context, passcode string) bool {
	rows, err := q.db.QueryContext(ctx, `SELECT passcode_hash FROM telive.host_users WHERE active = TRUE`)
	if err != nil {
		slog.Error("AuthenticateHost query", "err", err)
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			slog.Error("AuthenticateHost scan", "err", err)
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(passcode)) == nil {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("AuthenticateHost rows", "err", err)
	}
	return false
}
