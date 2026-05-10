# Frontend Testing Options

## Option 1: Playwright (end-to-end)

Spins up a real browser against the running server and drives it like a user would. Catches
exactly the kinds of bugs we've hit — CSP violations, broken event delegation, HTMX
interactions, inline style issues.

**Tradeoffs:** Requires Node.js, a running Go server, and a running database. Highest
confidence but most setup. Would likely want testcontainers to spin up Postgres the same
way the queue integration tests do.

---

## Option 2: Jest + jsdom (unit)

Tests JS functions in isolation — `toggleAddSong`, `renderResults`, `openRemoveDialog`,
drag handlers. No browser or server needed. Fast and lightweight.

**Tradeoffs:** jsdom doesn't enforce CSP, doesn't run real HTMX, and can't catch the class
of bug we've repeatedly hit (inline styles blocked by CSP, event delegation broken by
declaration order). Tests the logic inside functions, not the wiring between them.

---

## Option 3: Extend Go handler tests to assert on rendered HTML

The router tests already exercise HTML output at the HTTP level. Extending them to assert
on specific rendered attributes — e.g. that no `onclick=` appears in the queue partial, that
`data-name` is present and HTML-escaped — catches template-level XSS and attribute bugs
without any JS runtime.

**Tradeoffs:** Doesn't test JS behavior at all, only server-rendered HTML. Covers the
template surface cheaply.

---

## Recommendation

**Option 3** costs almost nothing and directly guards against the XSS class of bug.
**Playwright** would catch everything but is a meaningful commitment. **Jest/jsdom** has
poor signal-to-noise for this codebase because the JS is mostly DOM wiring rather than
pure logic.