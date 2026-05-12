# Signup Queue Ordering

New signups are inserted after the last person with 0 performances, pushing anyone who has already performed toward the end. The host can still reorder manually via drag-and-drop.

---

**Case 1: Fresh queue, no one has performed**

| Before | After Dan signs up |
|--------|-------------------|
| 1. Alice (0 performances) | 1. Alice (0 performances) |
| 2. Bob (0 performances) | 2. Bob (0 performances) |
| 3. Carol (0 performances) | 3. Carol (0 performances) |
| | 4. Dan (0 performances) |

Dan lands at the end. Same as before.

---

**Case 2: First singer has performed and been moved to bottom**

| Before | After Dan signs up |
|--------|-------------------|
| 1. Bob (0 performances) | 1. Bob (0 performances) |
| 2. Carol (0 performances) | 2. Carol (0 performances) |
| 3. Alice (1 performance) | 3. Dan (0 performances) |
| | 4. Alice (1 performance) |

Dan lands after Carol, before Alice.

---

**Case 3: Multiple people have performed**

| Before | After Dan signs up |
|--------|-------------------|
| 1. Carol (0 performances) | 1. Carol (0 performances) |
| 2. Alice (1 performance) | 2. Dan (0 performances) |
| 3. Bob (1 performance) | 3. Alice (1 performance) |
| | 4. Bob (1 performance) |

Dan lands after Carol, the only unperformed singer.

---

**Case 4: Everyone has performed**

| Before | After Dan signs up |
|--------|-------------------|
| 1. Alice (1 performance) | 1. Dan (0 performances) |
| 2. Bob (1 performance) | 2. Alice (1 performance) |
| 3. Carol (1 performance) | 3. Bob (1 performance) |
| | 4. Carol (1 performance) |

No unperformed anchor — Dan moves to the front since he's the only one who hasn't sung.

---

**Case 5: Someone was skipped (0 performances, not at front)**

| Before | After Dan signs up |
|--------|-------------------|
| 1. Carol (0 performances) | 1. Carol (0 performances) |
| 2. Bob (0 performances) ← skipped | 2. Bob (0 performances) |
| 3. Alice (1 performance) | 3. Dan (0 performances) |
| | 4. Alice (1 performance) |

Dan lands after Bob, the last unperformed person.

---

## Concurrent Signups

Two singers signing up at the same time could both read the same queue state, compute the same insertion position, and collide.

To prevent this, the `Add()` transaction will use `SELECT ... FOR UPDATE` when reading the current queue positions. This locks the rows for the duration of the transaction — a second concurrent signup that hits the same query will block until the first transaction commits. Once the first singer is inserted and the lock is released, the second transaction reads the updated queue (including the new entry) and computes a different position.

The result: concurrent signups are serialized at the database level. The order they land in the queue reflects the order their transactions committed, not the order they arrived.