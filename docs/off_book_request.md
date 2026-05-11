# Off-Book Request

Audience members can optionally signal that they're open to an off-book request when they get on stage. This appears as a visual indicator on their card in the host queue view.

---

## Schema

Add a boolean flag to `telive.signups`:

```sql
ALTER TABLE telive.signups ADD COLUMN off_book_request BOOLEAN NOT NULL DEFAULT FALSE;
```

No changes to `entry_songs` or `songs`. The song list stays clean.

---

## Signup Flow

Add an optional checkbox to the signup form:

> ☐ I'm open to an off-book request

The value is submitted with the form and stored in `signups.off_book_request`. Unchecked defaults to false.

---

## Data Layer

- Add `OffBookRequest bool` to the `Entry` struct in `queue/queue.go`
- Include `qe.off_book_request` in the `Entries()` SELECT query
- Scan it in `scanEntries`

No new queries needed.

---

## Host View

When `OffBookRequest` is true, render a small indicator on the entry card in `host_queue.html` — distinct from the song list, not a clickable action. Something like:

```
Alice
  ★ Open to off-book request
  • Bohemian Rhapsody — Queen  ✓
  • Wonderwall — Oasis
```

The indicator is display-only on the host side. No action required.

---

## What This Does Not Include

- No way for the host to mark the request as fulfilled — it's informational only
- No filtering or sorting by off-book availability
- No changes to the audience queue view (`/`)