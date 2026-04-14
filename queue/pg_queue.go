package queue

import (
	"database/sql"
	"log/slog"

	"github.com/iamdanhart/te-live/catalog"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// todayFilter is appended to WHERE clauses on signups and performed_songs
// to scope all queries to the current calendar day.
const todayQueueEntries = `created_at >= CURRENT_DATE`
const todayPerformed = `performed_at >= CURRENT_DATE`

// firstTodayID is a subquery that returns the id of the first entry in today's queue.
const firstTodayID = `(SELECT id FROM signups WHERE created_at >= CURRENT_DATE ORDER BY position ASC LIMIT 1)`

type PgQueue struct {
	db *sql.DB
}

func NewPgQueue(dsn string) (*PgQueue, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PgQueue{db: db}, nil
}

func (q *PgQueue) Entries() []Entry {
	rows, err := q.db.Query(`
		SELECT qe.id, qe.name, s.id, s.title, s.artist, s.tab_url, es.performed
		FROM signups qe
		JOIN entry_songs es ON es.entry_id = qe.id
		JOIN songs s ON s.id = es.song_id
		WHERE ` + todayQueueEntries + `
		ORDER BY qe.position ASC, es.sort_order ASC`)
	if err != nil {
		slog.Error("Entries query", "err", err)
		return nil
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (q *PgQueue) SignupsOpen() bool {
	var value string
	err := q.db.QueryRow(`SELECT value FROM settings WHERE key = 'signups_open'`).Scan(&value)
	if err != nil {
		slog.Error("SignupsOpen query", "err", err)
		return false
	}
	return value == "true"
}

func (q *PgQueue) ToggleSignups() bool {
	var value string
	err := q.db.QueryRow(`
		UPDATE settings
		SET value = CASE WHEN value = 'true' THEN 'false' ELSE 'true' END
		WHERE key = 'signups_open'
		RETURNING value`).Scan(&value)
	if err != nil {
		slog.Error("ToggleSignups query", "err", err)
		return false
	}
	return value == "true"
}

func (q *PgQueue) Add(name string, songIDs []int) error {
	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var entryID int
	err = tx.QueryRow(`
		INSERT INTO signups (name, position)
		VALUES ($1, COALESCE((SELECT MAX(position) FROM signups WHERE `+todayQueueEntries+`), 0) + 1)
		RETURNING id`, name).Scan(&entryID)
	if err != nil {
		return err
	}

	for i, songID := range songIDs {
		_, err = tx.Exec(`
			INSERT INTO entry_songs (entry_id, song_id, sort_order)
			VALUES ($1, $2, $3)`,
			entryID, songID, i)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (q *PgQueue) MoveCurrentToBottom() {
	_, err := q.db.Exec(`
		UPDATE signups
		SET position = (SELECT MAX(position) FROM signups WHERE ` + todayQueueEntries + `) + 1
		WHERE id = ` + firstTodayID)
	if err != nil {
		slog.Error("MoveCurrentToBottom", "err", err)
	}
}

func (q *PgQueue) RemoveCurrent() {
	_, err := q.db.Exec(`DELETE FROM signups WHERE id = ` + firstTodayID)
	if err != nil {
		slog.Error("RemoveCurrent", "err", err)
	}
}

func (q *PgQueue) CompleteCurrentSong(singer string, songID int) {
	tx, err := q.db.Begin()
	if err != nil {
		slog.Error("CompleteCurrentSong begin tx", "err", err)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE entry_songs SET performed = true
		WHERE entry_id = `+firstTodayID+`
		  AND song_id = $1`,
		songID)
	if err != nil {
		slog.Error("CompleteCurrentSong mark performed", "err", err)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO performed_songs (singer, song_id)
		VALUES ($1, $2)`,
		singer, songID)
	if err != nil {
		slog.Error("CompleteCurrentSong insert performed_songs", "err", err)
		return
	}

	_, err = tx.Exec(`
		UPDATE signups SET times_on_stage = times_on_stage + 1
		WHERE id = ` + firstTodayID)
	if err != nil {
		slog.Error("CompleteCurrentSong increment times_on_stage", "err", err)
		return
	}

	if err := tx.Commit(); err != nil {
		slog.Error("CompleteCurrentSong commit", "err", err)
	}
}

func (q *PgQueue) Performed() []PerformedSong {
	rows, err := q.db.Query(`
		SELECT ps.singer, s.id, s.title, s.artist, s.tab_url
		FROM performed_songs ps
		JOIN songs s ON s.id = ps.song_id
		WHERE ` + todayPerformed + `
		ORDER BY ps.performed_at ASC`)
	if err != nil {
		slog.Error("Performed query", "err", err)
		return nil
	}
	defer rows.Close()

	var result []PerformedSong
	for rows.Next() {
		var ps PerformedSong
		if err := rows.Scan(&ps.Singer, &ps.Song.ID, &ps.Song.Title, &ps.Song.Artist, &ps.Song.TabUrl); err != nil {
			slog.Error("Performed scan", "err", err)
			continue
		}
		result = append(result, ps)
	}
	return result
}

func (q *PgQueue) AddSongToFirst(songID int) {
	_, err := q.db.Exec(`
		INSERT INTO entry_songs (entry_id, song_id, sort_order)
		VALUES (
			`+firstTodayID+`,
			$1,
			COALESCE((SELECT MAX(sort_order) FROM entry_songs WHERE entry_id = `+firstTodayID+`), 0) + 1
		)`, songID)
	if err != nil {
		slog.Error("AddSongToFirst", "err", err)
	}
}

func (q *PgQueue) HasName(name string) bool {
	var exists bool
	err := q.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM signups
			WHERE LOWER(name) = LOWER($1) AND `+todayQueueEntries+`
		)`, name).Scan(&exists)
	if err != nil {
		slog.Error("HasName query", "err", err)
		return false
	}
	return exists
}

func (q *PgQueue) MoveEntry(id, afterID int) {
	rows, err := q.db.Query(`
		SELECT id, position FROM signups
		WHERE `+todayQueueEntries+`
		ORDER BY position ASC`)
	if err != nil {
		slog.Error("MoveEntry query", "err", err)
		return
	}
	defer rows.Close()

	type row struct {
		id  int
		pos float64
	}
	var entries []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.pos); err != nil {
			slog.Error("MoveEntry scan", "err", err)
			return
		}
		entries = append(entries, r)
	}

	var newPos float64
	if afterID == 0 {
		if len(entries) == 0 {
			newPos = 1
		} else {
			newPos = entries[0].pos - 1
		}
	} else {
		afterIdx := -1
		for i, e := range entries {
			if e.id == afterID {
				afterIdx = i
				break
			}
		}
		if afterIdx == -1 {
			slog.Error("MoveEntry afterID not found", "afterID", afterID)
			return
		}
		if afterIdx == len(entries)-1 {
			newPos = entries[afterIdx].pos + 1
		} else {
			newPos = (entries[afterIdx].pos + entries[afterIdx+1].pos) / 2
		}
	}

	if _, err := q.db.Exec(`UPDATE signups SET position = $1 WHERE id = $2`, newPos, id); err != nil {
		slog.Error("MoveEntry update", "err", err)
	}
}

// scanEntries collapses the joined rows into []Entry, grouping songs by singer.
func scanEntries(rows *sql.Rows) []Entry {
	var entries []Entry
	entryIndex := map[int]int{} // queue_entry id → index in entries slice

	for rows.Next() {
		var (
			entryID   int
			name      string
			songID    int
			title     string
			artist    string
			tabUrl    string
			performed bool
		)
		if err := rows.Scan(&entryID, &name, &songID, &title, &artist, &tabUrl, &performed); err != nil {
			slog.Error("scanEntries scan", "err", err)
			continue
		}
		idx, exists := entryIndex[entryID]
		if !exists {
			entries = append(entries, Entry{ID: entryID, Name: name})
			idx = len(entries) - 1
			entryIndex[entryID] = idx
		}
		entries[idx].Songs = append(entries[idx].Songs, SongEntry{
			Song:      catalog.Song{ID: songID, Title: title, Artist: artist, TabUrl: tabUrl},
			Performed: performed,
		})
	}
	return entries
}