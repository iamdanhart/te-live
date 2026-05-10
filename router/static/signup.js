// Toggle a song row as selected when clicked. Caps selection at 3 songs.
// When selected, the row moves to the top section; when deselected, it
// returns to its original alphabetical position. The search box is also
// cleared so the full unselected list is visible after picking a song.
document.querySelectorAll('.catalog-row').forEach(row => {
    row.addEventListener('click', function() {
        const alreadySelected = this.classList.contains('selected');
        if (!alreadySelected && document.querySelectorAll('.catalog-row.selected').length >= 3) {
            return;
        }
        this.classList.toggle('selected');
        const atLimit = document.querySelectorAll('.catalog-row.selected').length >= 3;
        // at-limit grays out the unselected list via CSS so no more picks are possible
        document.getElementById('catalog-wrapper').classList.toggle('at-limit', atLimit);
        reorderRows(this);
        updateSongInputs();
        updateSubmit();
        document.getElementById('search').value = '';
        document.querySelectorAll('#unselected-rows .catalog-row').forEach(r => r.style.display = '');
    });
});

// Move a row into the selected or unselected tbody. For deselected rows,
// data-index (the original catalog order) is used to find the right slot.
function reorderRows(row) {
    if (row.classList.contains('selected')) {
        document.getElementById('selected-rows').appendChild(row);
    } else {
        const tbody = document.getElementById('unselected-rows');
        const index = parseInt(row.dataset.index);
        const insertBefore = [...tbody.querySelectorAll('.catalog-row')]
            .find(r => parseInt(r.dataset.index) > index);
        tbody.insertBefore(row, insertBefore || null);
    }
}

// Keep the hidden song inputs in sync with the selected rows so they are
// included when the form is submitted.
function updateSongInputs() {
    const container = document.getElementById('song-inputs');
    const rows = [...document.querySelectorAll('.catalog-row.selected')];
    container.innerHTML = rows
        .map(row => `<input type="hidden" name="song" value="${row.dataset.id}">`)
        .join('');
}

let signupSucceeded = false;

// The form targets #status via hx-target, so the server's response fragment
// is swapped in there. We listen for that swap to open the result modal.
// Filtering by target id prevents the name-check htmx swap from also
// triggering the modal.
document.body.addEventListener('htmx:afterSwap', function(evt) {
    if (evt.detail.target.id !== 'status') return;
    signupSucceeded = true;
    const modal = document.getElementById('status-modal');
    if (!modal.open) modal.showModal();
});

// The rate limiter returns 429 when the cooldown hasn't elapsed. HTMX
// treats 4xx as an error and won't swap, so we handle it manually here.
document.body.addEventListener('htmx:responseError', function(evt) {
    if (evt.detail.xhr.status === 429) {
        signupSucceeded = false;
        document.getElementById('status').innerHTML = '<p>Too many submissions. Please wait before trying again.</p>';
        const modal = document.getElementById('status-modal');
        if (!modal.open) modal.showModal();
    }
});

// Clear the status content when the modal closes so stale messages
// don't flash briefly before HTMX overwrites them on the next submission.
document.getElementById('status-modal').addEventListener('close', function() {
    document.getElementById('status').innerHTML = '';
});

// On success, send the user to the public queue view. On failure, just
// close the modal so they can try again.
document.getElementById('modal-ok').addEventListener('click', function() {
    if (signupSucceeded) {
        window.location = '/';
    } else {
        document.getElementById('status-modal').close();
    }
});

// Disable the submit button until both a name and at least one song are present.
function updateSubmit() {
    const hasName = document.getElementById('name').value.trim() !== '';
    const hasSong = document.querySelectorAll('.catalog-row.selected').length > 0;
    document.getElementById('submit-btn').disabled = !(hasName && hasSong);
}

document.getElementById('name').addEventListener('input', updateSubmit);

document.querySelector('form').addEventListener('htmx:beforeRequest', function(evt) {
    if (evt.detail.elt !== this) return;
    const btn = document.getElementById('submit-btn');
    btn.disabled = true;
    btn.textContent = 'Submitting…';
});

document.querySelector('form').addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.elt !== this) return;
    document.getElementById('submit-btn').textContent = 'Sign Up';
    updateSubmit();
});

// Column header clicks toggle sort direction; clicking a new column resets to ascending.
let sortCol = null, sortDir = 1;

document.querySelectorAll('th[data-col]').forEach(th => {
    th.addEventListener('click', function() {
        const col = this.dataset.col;
        if (sortCol === col) {
            sortDir *= -1;
        } else {
            sortCol = col;
            sortDir = 1;
        }
        document.querySelectorAll('th[data-col]').forEach(h => {
            h.classList.toggle('sorted', h === this);
            h.textContent = h.dataset.col.charAt(0).toUpperCase() + h.dataset.col.slice(1);
            if (h === this) h.textContent += sortDir === 1 ? ' ↑' : ' ↓';
        });
        const tbody = document.getElementById('unselected-rows');
        [...tbody.querySelectorAll('.catalog-row')]
            .sort((a, b) => a.dataset[col].localeCompare(b.dataset[col]) * sortDir)
            .forEach(row => tbody.appendChild(row));
    });
});

// Hide rows that don't match the search query. Clearing the input restores all rows.
document.getElementById('search').addEventListener('input', function() {
    const query = this.value.toLowerCase();
    document.querySelectorAll('#unselected-rows .catalog-row').forEach(row => {
        row.style.display = row.textContent.toLowerCase().includes(query) ? '' : 'none';
    });
});