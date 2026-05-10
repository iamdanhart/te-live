let songCatalog = null;
let filteredResults = [];

function toggleAddSong() {
    const picker = document.getElementById('add-song-picker');
    const isHidden = picker.style.display === 'none' || picker.style.display === '';
    picker.style.display = isHidden ? 'block' : 'none';
    if (isHidden) {
        if (!songCatalog) {
            fetch('/catalog').then(r => r.json()).then(data => {
                songCatalog = data.songs;
                renderResults('');
            });
        } else {
            renderResults(document.getElementById('add-song-search').value);
        }
        document.getElementById('add-song-search').focus();
    }
}

function renderResults(query) {
    const q = query.toLowerCase();
    filteredResults = songCatalog.filter(s =>
        s.title.toLowerCase().includes(q) || s.artist.toLowerCase().includes(q)
    );
    const ul = document.getElementById('add-song-results');
    ul.innerHTML = '';
    // Build DOM nodes instead of HTML strings so song data is never interpreted as markup
    filteredResults.forEach((s, i) => {
        const li = document.createElement('li');
        li.className = 'add-song-result';
        li.onclick = () => selectSong(i);
        const title = document.createElement('span');
        title.textContent = s.title;
        const artist = document.createTextNode(' — ' + s.artist);
        li.appendChild(title);
        li.appendChild(artist);
        ul.appendChild(li);
    });
}

function openRemoveDialog(name) {
    document.getElementById('remove-name').textContent = name;
    document.getElementById('remove-dialog').showModal();
}

function confirmRemove() {
    document.getElementById('remove-dialog').close();
    htmx.ajax('POST', '/host/remove', {
        target: '#queue-list',
        swap: 'innerHTML'
    });
}

// Pause the queue poll while the add-song picker or remove dialog is open
// so the swap doesn't destroy them mid-interaction.
document.body.addEventListener('htmx:beforeRequest', function(evt) {
    if (evt.detail.elt.id === 'queue-list') {
        const picker = document.getElementById('add-song-picker');
        const dialog = document.getElementById('remove-dialog');
        if ((picker && picker.style.display === 'block') || (dialog && dialog.open)) {
            evt.preventDefault();
        }
    }
});

function toggleSignups() {
    fetch('/signups/toggle', {method: 'POST'})
        .then(r => r.json())
        .then(data => {
            const btn = document.getElementById('signup-toggle-btn');
            const open = data.signups_open;
            btn.textContent = open ? 'Close Signups' : 'Open Signups';
            btn.classList.toggle('open', open);
        });
}

document.getElementById('signup-toggle-btn').addEventListener('click', toggleSignups);

document.querySelector('.dialog-cancel-btn').addEventListener('click', function() {
    document.getElementById('remove-dialog').close();
});

document.querySelector('.dialog-confirm-btn').addEventListener('click', confirmRemove);

let draggedId = null;
const queueList = document.getElementById('queue-list');

queueList.addEventListener('click', function(e) {
    if (e.target.closest('.remove-btn')) {
        openRemoveDialog(e.target.closest('.remove-btn').dataset.name);
        return;
    }
    if (e.target.closest('.add-song-btn')) {
        toggleAddSong();
    }
});

queueList.addEventListener('input', function(e) {
    if (e.target.id === 'add-song-search') {
        renderResults(e.target.value);
    }
});

queueList.addEventListener('dragstart', function(e) {
    const entry = e.target.closest('[data-id]');
    if (!entry) return;
    draggedId = entry.dataset.id;
    e.dataTransfer.effectAllowed = 'move';
});

queueList.addEventListener('dragover', function(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    const entry = e.target.closest('[data-id]');
    queueList.querySelectorAll('[data-id]').forEach(el => el.classList.remove('drag-over-top', 'drag-over-bottom'));
    if (!entry || entry.dataset.id === draggedId) return;
    const rect = entry.getBoundingClientRect();
    if (e.clientY < rect.top + rect.height / 2) {
        entry.classList.add('drag-over-top');
    } else {
        entry.classList.add('drag-over-bottom');
    }
});

queueList.addEventListener('dragleave', function(e) {
    if (!queueList.contains(e.relatedTarget)) {
        queueList.querySelectorAll('[data-id]').forEach(el => el.classList.remove('drag-over-top', 'drag-over-bottom'));
    }
});

queueList.addEventListener('drop', function(e) {
    e.preventDefault();
    const target = e.target.closest('[data-id]');
    queueList.querySelectorAll('[data-id]').forEach(el => el.classList.remove('drag-over-top', 'drag-over-bottom'));
    if (!target || target.dataset.id === draggedId) return;
    const rect = target.getBoundingClientRect();
    let afterId;
    if (e.clientY < rect.top + rect.height / 2) {
        const prev = target.previousElementSibling;
        afterId = (prev && prev.dataset.id) ? prev.dataset.id : '0';
    } else {
        afterId = target.dataset.id;
    }
    htmx.ajax('POST', '/host/move', {
        target: '#queue-list',
        swap: 'innerHTML',
        values: { id: draggedId, after_id: afterId }
    });
});

function selectSong(i) {
    const song = filteredResults[i];
    htmx.ajax('POST', '/host/add-song', {
        target: '#queue-list',
        swap: 'innerHTML',
        values: {song_id: song.id}
    });
    document.getElementById('add-song-picker').style.display = 'none';
    document.getElementById('add-song-search').value = '';
}