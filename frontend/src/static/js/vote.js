// -----------------------------------------------------------------------
// Constants (TOTAL_POINTS, SERVER_REMAINING, SERVER_VOTES_CAST,
// OUT_OF_POINTS) are injected by the template as globals before this
// script is loaded.
// -----------------------------------------------------------------------

// -----------------------------------------------------------------------
// State
// -----------------------------------------------------------------------
// pendingAllocation: songId (number) → points (number) the user is
// currently setting in THIS session (before submitting).
// This is separate from SERVER_VOTES_CAST which is already committed.
const pendingAllocation = {};

// Budget remaining for pending actions only.
let localRemaining = SERVER_REMAINING;

// -----------------------------------------------------------------------
// Boot: initialise stepper states
// -----------------------------------------------------------------------
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.stepper-plus').forEach(btn => {
        btn.addEventListener('click', () => stepPoints(Number(btn.dataset.songId), +1));
    });
    document.querySelectorAll('.stepper-minus').forEach(btn => {
        btn.addEventListener('click', () => stepPoints(Number(btn.dataset.songId), -1));
    });

    refreshBudgetUI();
    refreshBasket();
});

// -----------------------------------------------------------------------
// Core: adjust pending points for a song by delta (+1 / -1)
// -----------------------------------------------------------------------
function stepPoints(songId, delta) {
    const current = pendingAllocation[songId] || 0;
    const newVal  = current + delta;

    if (newVal < 0) return;
    if (delta > 0 && localRemaining <= 0) return;

    pendingAllocation[songId] = newVal;
    localRemaining -= delta;

    updateCardDisplay(songId, newVal);
    refreshBudgetUI();
    refreshBasket();
}

// -----------------------------------------------------------------------
// Update a single card's stepper display
// -----------------------------------------------------------------------
function updateCardDisplay(songId, pts) {
    const display = document.getElementById('pts-display-' + songId);
    if (!display) return;

    display.textContent = pts;
    display.className = display.className.replace(/text-esc-(yellow|muted)/g, '');
    display.classList.add(pts > 0 ? 'text-esc-yellow' : 'text-esc-muted');

    const card = document.querySelector(`.country-card[data-song-id="${songId}"]`);
    if (card) {
        if (pts > 0) {
            card.classList.add('card-allocated');
        } else {
            const serverPts = Number(SERVER_VOTES_CAST[String(songId)] || 0);
            if (serverPts === 0) card.classList.remove('card-allocated');
        }
    }

    const minusBtn = document.querySelector(`.stepper-minus[data-song-id="${songId}"]`);
    if (minusBtn) minusBtn.disabled = (pts <= 0);

    refreshPlusButtons();
}

// -----------------------------------------------------------------------
// Enable / disable all + buttons based on remaining budget
// -----------------------------------------------------------------------
function refreshPlusButtons() {
    document.querySelectorAll('.stepper-plus').forEach(btn => {
        btn.disabled = (localRemaining <= 0);
    });
}

// -----------------------------------------------------------------------
// Update the top budget bar + labels
// -----------------------------------------------------------------------
function refreshBudgetUI() {
    const usedTotal    = TOTAL_POINTS - SERVER_REMAINING;
    const pendingTotal = Object.values(pendingAllocation).reduce((a, b) => a + b, 0);
    const displayUsed  = usedTotal + pendingTotal;
    const displayLeft  = TOTAL_POINTS - displayUsed;

    const bar   = document.getElementById('budget-bar');
    const label = document.getElementById('budget-remaining');
    const used  = document.getElementById('budget-used-label');

    if (bar)   bar.style.width   = Math.min(100, (displayUsed / TOTAL_POINTS) * 100) + '%';
    if (label) label.textContent = Math.max(0, displayLeft);
    if (used)  used.textContent  = displayUsed + ' used';

    if (bar) {
        bar.classList.remove('budget-pulse');
        void bar.offsetWidth;
        bar.classList.add('budget-pulse');
    }
}

// -----------------------------------------------------------------------
// Basket: show/hide sticky bar, populate tags
// -----------------------------------------------------------------------
function refreshBasket() {
    const basket    = document.getElementById('vote-basket');
    const tagsEl    = document.getElementById('basket-tags');
    const pendingEl = document.getElementById('basket-pending-count');

    const entries = Object.entries(pendingAllocation).filter(([, pts]) => pts > 0);

    if (entries.length === 0) {
        basket.classList.add('hidden');
        return;
    }

    basket.classList.remove('hidden');

    const totalPending = entries.reduce((sum, [, pts]) => sum + pts, 0);
    if (pendingEl) pendingEl.textContent = totalPending;

    tagsEl.innerHTML = entries.map(([songId, pts]) => {
        const card        = document.querySelector(`.country-card[data-song-id="${songId}"]`);
        const countryName = card ? card.dataset.countryName : `Song ${songId}`;
        const countryId   = card ? card.dataset.countryId   : '';
        return `
            <span class="inline-flex items-center gap-1.5 rounded-full border border-esc-yellow/30 bg-esc-yellow/10 px-2.5 py-1 text-[11px] text-esc-yellow font-medium">
                <img src="https://flagfeed.com/flags/${countryId}" alt="" class="w-3.5 h-3.5 rounded-full object-cover shrink-0">
                ${countryName}
                <span class="font-bold">${pts}pt${pts !== 1 ? 's' : ''}</span>
            </span>`;
    }).join('');
}

// -----------------------------------------------------------------------
// Clear all pending allocations
// -----------------------------------------------------------------------
function clearBasket() {
    Object.keys(pendingAllocation).forEach(songId => {
        pendingAllocation[songId] = 0;
        updateCardDisplay(Number(songId), 0);
    });
    localRemaining = SERVER_REMAINING;
    refreshBudgetUI();
    refreshBasket();
}

// -----------------------------------------------------------------------
// Submit modal: open / close / populate
// -----------------------------------------------------------------------
function openSubmitModal() {
    const entries = Object.entries(pendingAllocation).filter(([, pts]) => pts > 0);
    if (entries.length === 0) return;

    const listEl = document.getElementById('modal-allocation-list');
    listEl.innerHTML = entries.map(([songId, pts]) => {
        const card        = document.querySelector(`.country-card[data-song-id="${songId}"]`);
        const countryName = card ? card.dataset.countryName : `Song ${songId}`;
        const songName    = card ? card.dataset.songName    : '';
        const countryId   = card ? card.dataset.countryId   : '';
        return `
            <div class="flex items-center gap-2.5 rounded-lg bg-esc-surface/40 px-3 py-2">
                <img src="https://flagfeed.com/flags/${countryId}" alt="" class="w-5 h-5 rounded-full object-cover shrink-0">
                <div class="flex-1 min-w-0">
                    <p class="text-xs font-medium text-esc-text truncate">${countryName}</p>
                    <p class="text-[10px] text-esc-muted truncate">${songName}</p>
                </div>
                <span class="font-heading font-bold text-sm text-esc-yellow shrink-0">${pts} pt${pts !== 1 ? 's' : ''}</span>
            </div>`;
    }).join('');

    document.getElementById('submit-form').reset();
    document.getElementById('submit-response').classList.add('hidden');
    document.getElementById('submit-progress').classList.add('hidden');
    document.getElementById('submit-log').classList.add('hidden');
    document.getElementById('submit-log').innerHTML = '';
    document.getElementById('submit-btn').disabled = false;
    document.getElementById('submit-btn-text').textContent = 'Submit All Votes';
    document.getElementById('submit-spinner').classList.add('hidden');

    document.getElementById('submit-modal').classList.remove('hidden');
    document.body.style.overflow = 'hidden';
}

function closeSubmitModal() {
    document.getElementById('submit-modal').classList.add('hidden');
    document.body.style.overflow = '';
}

document.addEventListener('keydown', e => {
    if (e.key === 'Escape') closeSubmitModal();
});

// -----------------------------------------------------------------------
// Form submission: send one POST per song sequentially
// -----------------------------------------------------------------------
document.getElementById('submit-form').addEventListener('submit', async function(e) {
    e.preventDefault();

    if (typeof window.hasVoteCookieConsent === 'function' && !window.hasVoteCookieConsent()) {
        showResponse('error', 'Please accept required vote cookies before submitting votes.');
        if (typeof window.revealCookieBanner === 'function') {
            window.revealCookieBanner();
        }
        return;
    }

    const phone      = document.getElementById('phone-number').value.trim();
    const ownCountry = document.getElementById('own-country').value;

    if (!phone || !ownCountry) {
        showResponse('error', 'Please fill in your country and phone number.');
        return;
    }

    const entries = Object.entries(pendingAllocation).filter(([, pts]) => pts > 0);
    if (entries.length === 0) {
        showResponse('error', 'No points allocated.');
        return;
    }

    const btn     = document.getElementById('submit-btn');
    const btnText = document.getElementById('submit-btn-text');
    const spinner = document.getElementById('submit-spinner');
    btn.disabled  = true;
    spinner.classList.remove('hidden');
    btnText.textContent = 'Submitting…';

    const progressWrap = document.getElementById('submit-progress');
    const progressBar  = document.getElementById('submit-progress-bar');
    const progressLbl  = document.getElementById('submit-progress-label');
    const progressCnt  = document.getElementById('submit-progress-count');
    const logEl        = document.getElementById('submit-log');

    progressWrap.classList.remove('hidden');
    logEl.classList.remove('hidden');
    logEl.innerHTML = '';

    const total   = entries.length;
    let succeeded = 0;
    let failed    = 0;
    const errors  = [];

    for (let i = 0; i < entries.length; i++) {
        const [songId, pts] = entries[i];
        const card        = document.querySelector(`.country-card[data-song-id="${songId}"]`);
        const countryName = card ? card.dataset.countryName : `Song ${songId}`;

        progressLbl.textContent = `Submitting vote for ${countryName}…`;
        progressCnt.textContent = `${i + 1} / ${total}`;
        progressBar.style.width = ((i / total) * 100) + '%';

        const formData = new FormData();
        formData.append('songID',     songId);
        formData.append('phoneNum',   phone);
        formData.append('ownCountry', ownCountry);
        formData.append('points',     String(pts));

        try {
            const resp = await fetch('/vote/submit', { method: 'POST', body: formData });
            const data = await resp.json();

            if (resp.ok) {
                succeeded++;
                logEl.innerHTML += logEntry('success', `✓ ${countryName}: ${pts} pt${pts !== 1 ? 's' : ''} cast`);
                if (typeof data.votes_remaining === 'number') {
                    localRemaining = data.votes_remaining;
                }
            } else {
                failed++;
                const msg = data.error || data.message || 'Unknown error';
                errors.push(`${countryName}: ${msg}`);
                logEl.innerHTML += logEntry('error', `✗ ${countryName}: ${msg}`);
            }
        } catch (err) {
            failed++;
            errors.push(`${countryName}: network error`);
            logEl.innerHTML += logEntry('error', `✗ ${countryName}: network error`);
        }

        logEl.scrollTop = logEl.scrollHeight;
    }

    progressBar.style.width = '100%';
    progressLbl.textContent = 'Done!';
    progressCnt.textContent = `${total} / ${total}`;

    spinner.classList.add('hidden');

    if (failed === 0) {
        btnText.textContent = '✓ All Votes Submitted!';
        showResponse('success', `All ${succeeded} vote${succeeded !== 1 ? 's' : ''} submitted successfully! Refreshing…`);
        setTimeout(() => { closeSubmitModal(); window.location.reload(); }, 2000);
    } else if (succeeded > 0) {
        btnText.textContent = 'Partially Submitted';
        showResponse('warning', `${succeeded} vote${succeeded !== 1 ? 's' : ''} submitted, ${failed} failed. Refreshing…`);
        setTimeout(() => { closeSubmitModal(); window.location.reload(); }, 3000);
    } else {
        btnText.textContent = 'Submission Failed';
        btn.disabled = false;
        showResponse('error', errors[0] || 'All votes failed. Please try again.');
    }
});

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------
function logEntry(type, text) {
    const colour = type === 'success' ? 'text-green-400' : 'text-red-400';
    return `<p class="${colour}">${escHtml(text)}</p>`;
}

function escHtml(str) {
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function showResponse(type, message) {
    const el = document.getElementById('submit-response');
    el.classList.remove('hidden');
    const styles = {
        success: 'border border-green-500/30 bg-green-500/10 text-green-400',
        error:   'border border-red-500/30 bg-red-500/10 text-red-400',
        warning: 'border border-esc-yellow/30 bg-esc-yellow/10 text-esc-yellow',
    };
    el.className  = 'rounded-lg px-4 py-3 text-sm text-center ' + (styles[type] || styles.error);
    el.textContent = message;
}
