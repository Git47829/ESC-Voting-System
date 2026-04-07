// -----------------------------------------------------------------------
// Constants (votedSongs, totalSongs, usedPointValues) are injected by
// the template as globals before this script is loaded.
// -----------------------------------------------------------------------

const selectedPoints = {};

/**
 * Disable the given point value button across ALL song cards and
 * mark it with a strikethrough title so the jury member knows why.
 */
function disablePointValueGlobally(points) {
    usedPointValues.add(points);

    const legendOverlay = document.getElementById('legend-used-' + points);
    if (legendOverlay) {
        legendOverlay.classList.remove('hidden');
        legendOverlay.classList.add('flex');
        const badge = legendOverlay.previousElementSibling;
        if (badge) badge.classList.add('opacity-40');
    }

    document.querySelectorAll('.point-btn[data-points="' + points + '"]').forEach(function (btn) {
        const songId = parseInt(btn.dataset.songId);

        if (!votedSongs.has(songId) && selectedPoints[songId] === points) {
            delete selectedPoints[songId];

            const display = document.getElementById('selected-display-' + songId);
            if (display) {
                display.classList.add('hidden');
                display.classList.remove('flex');
            }

            const submitBtn = document.getElementById('submit-btn-' + songId);
            if (submitBtn) submitBtn.disabled = true;

            btn.classList.remove('selected');
        }

        btn.disabled = true;
        btn.classList.add('opacity-40', 'cursor-not-allowed');
        btn.title = points + ' pts already awarded to another entry';
    });
}

function selectPoints(btn, songId, points) {
    if (usedPointValues.has(points)) return;

    const group   = document.getElementById('points-group-' + songId);
    const buttons = group.querySelectorAll('.point-btn');
    buttons.forEach(function (b) { b.classList.remove('selected'); });

    btn.classList.add('selected');
    selectedPoints[songId] = points;

    const display = document.getElementById('selected-display-' + songId);
    const valueEl = document.getElementById('selected-value-' + songId);
    display.classList.remove('hidden');
    display.classList.add('flex');
    valueEl.textContent = points;

    if (!votedSongs.has(songId)) {
        const submitBtn = document.getElementById('submit-btn-' + songId);
        submitBtn.disabled = false;
    }
}

async function submitJuryVote(songId) {
    const points      = selectedPoints[songId];
    if (!points) return;

    const submitBtn   = document.getElementById('submit-btn-' + songId);
    const submitText  = document.getElementById('submit-text-' + songId);
    const responseDiv = document.getElementById('response-' + songId);
    const card        = document.getElementById('jury-card-' + songId);

    submitBtn.disabled       = true;
    submitText.textContent   = 'Sending…';

    const formData = new FormData();
    formData.append('songID', songId);
    formData.append('points', points);

    try {
        const resp = await fetch('/jury/submit', { method: 'POST', body: formData });
        const data = await resp.json();

        responseDiv.classList.remove('hidden');

        if (resp.ok) {
            responseDiv.className  = 'border-t border-green-500/20 px-5 py-3 text-sm text-green-400 bg-green-500/5';
            responseDiv.textContent = '✓ ' + points + ' points awarded successfully!';

            votedSongs.add(songId);
            card.classList.add('voted');

            const votedIcon = document.getElementById('voted-icon-' + songId);
            votedIcon.classList.remove('hidden');
            votedIcon.classList.add('vote-success-anim');

            submitBtn.classList.add('hidden');

            const group = document.getElementById('points-group-' + songId);
            group.querySelectorAll('.point-btn').forEach(function (b) {
                b.disabled = true;
                b.classList.add('opacity-40', 'cursor-not-allowed');
            });

            disablePointValueGlobally(points);
            updateVoteCount();

            setTimeout(function () { responseDiv.classList.add('hidden'); }, 3000);

        } else if (resp.status === 409) {
            disablePointValueGlobally(points);
            delete selectedPoints[songId];
            const display = document.getElementById('selected-display-' + songId);
            display.classList.add('hidden');
            responseDiv.className  = 'border-t border-red-500/20 px-5 py-3 text-sm text-red-400 bg-red-500/5';
            responseDiv.textContent = data.error || 'That point value has already been used.';
            submitBtn.classList.add('hidden');
            submitText.textContent  = 'Vote';
        } else {
            responseDiv.className  = 'border-t border-red-500/20 px-5 py-3 text-sm text-red-400 bg-red-500/5';
            responseDiv.textContent = data.error || data.message || 'Failed to submit vote.';
            submitBtn.disabled      = false;
            submitText.textContent  = 'Vote';
        }
    } catch (err) {
        responseDiv.classList.remove('hidden');
        responseDiv.className  = 'border-t border-red-500/20 px-5 py-3 text-sm text-red-400 bg-red-500/5';
        responseDiv.textContent = 'Network error — please try again.';
        submitBtn.disabled      = false;
        submitText.textContent  = 'Vote';
    }
}

function updateVoteCount() {
    const countEl    = document.getElementById('votes-cast-count');
    const allDoneMsg = document.getElementById('all-done-msg');
    countEl.textContent = votedSongs.size;

    if (votedSongs.size >= totalSongs) {
        allDoneMsg.classList.remove('hidden');
        allDoneMsg.classList.add('flex');
    }
}

// On page load, apply tooltip titles to all already-used/voted point buttons
document.addEventListener('DOMContentLoaded', function () {
    usedPointValues.forEach(function (points) {
        document.querySelectorAll('.point-btn[data-points="' + points + '"]').forEach(function (btn) {
            btn.title = points + ' pts already awarded to another entry';
        });
    });
    updateVoteCount();
});
