// -----------------------------------------------------------------------
// State
// -----------------------------------------------------------------------
let ws = null;
let reconnectDelay = 1000;
let hasReceivedData = false;

// -----------------------------------------------------------------------
// UI helpers
// -----------------------------------------------------------------------
function setStatus(connected) {
    const pingDot    = document.getElementById('ping-dot');
    const pingSolid  = document.getElementById('ping-dot-solid');
    const statusText = document.getElementById('status-text');
    const errorState = document.getElementById('error-state');

    if (connected) {
        pingDot.classList.remove('bg-esc-muted');
        pingDot.classList.add('animate-ping', 'bg-green-400');
        pingSolid.classList.remove('bg-esc-muted');
        pingSolid.classList.add('bg-green-400');
        statusText.textContent = 'Live';
        errorState.classList.add('hidden');
    } else {
        pingDot.classList.remove('animate-ping', 'bg-green-400');
        pingDot.classList.add('bg-esc-muted');
        pingSolid.classList.remove('bg-green-400');
        pingSolid.classList.add('bg-esc-muted');
        statusText.textContent = 'Reconnecting\u2026';
        if (hasReceivedData) {
            errorState.classList.remove('hidden');
        }
    }
}

function renderStats(msg) {
    const loadingEl   = document.getElementById('loading-state');
    const emptyEl     = document.getElementById('empty-state');
    const containerEl = document.getElementById('charts-container');

    loadingEl.classList.add('hidden');
    document.getElementById('vote-count').textContent = Number.isFinite(msg.vote_count) ? msg.vote_count : 0;
    hasReceivedData = true;

    if (!msg.charts || msg.vote_count === 0) {
        emptyEl.classList.remove('hidden');
        containerEl.classList.add('hidden');
        return;
    }

    emptyEl.classList.add('hidden');
    containerEl.classList.remove('hidden');

    const safeImageSrc = (value) => {
        if (typeof value !== 'string') return '';
        if (/^data:image\/(png|jpeg|gif|webp|svg\+xml);base64,[A-Za-z0-9+/=]+$/.test(value)) return value;
        return '';
    };

    document.getElementById('chart-voters-by-country').src =
        safeImageSrc(msg.charts.voters_by_country);
    document.getElementById('chart-votes-received').src =
        safeImageSrc(msg.charts.votes_received_by_country);
}

// -----------------------------------------------------------------------
// WebSocket
// -----------------------------------------------------------------------
function connect() {
    const wsUrl = `wss://${window.location.host}/eurostats/ws/stats`;
    ws = new WebSocket(wsUrl);

    ws.onopen = function () {
        reconnectDelay = 1000;
        setStatus(true);
    };

    ws.onmessage = function (event) {
        let msg;
        try {
            msg = JSON.parse(event.data);
        } catch (e) {
            return;
        }
        if (msg.type === 'ping') return;
        if (msg.type === 'stats') renderStats(msg);
    };

    ws.onclose = function () {
        setStatus(false);
        setTimeout(connect, reconnectDelay);
        reconnectDelay = Math.min(reconnectDelay * 2, 30000);
    };

    ws.onerror = function () {
        ws.close();
    };
}

// -----------------------------------------------------------------------
// Boot
// -----------------------------------------------------------------------
connect();
