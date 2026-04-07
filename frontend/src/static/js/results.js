// ---------------------------------------------------------------------------
// Flag helpers
// ---------------------------------------------------------------------------

function getFlagUrl(countryId) {
    const code = countryId ? countryId.slice(0, 2).toLowerCase() : null;
    return code
        ? { url: "https://flagfeed.com/flags/" + code, code }
        : null;
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------
let paused = false;
let secondsLeft = 10;
let pollInterval = null;
let countdownInterval = null;
let previousData = null;

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------
function renderResults(results) {
    const loadingEl   = document.getElementById("loading-state");
    const emptyEl     = document.getElementById("empty-state");
    const containerEl = document.getElementById("results-container");
    const errorEl     = document.getElementById("error-state");

    loadingEl.classList.add("hidden");
    errorEl.classList.add("hidden");

    if (!results || results.length === 0) {
        emptyEl.classList.remove("hidden");
        containerEl.classList.add("hidden");
        return;
    }
    emptyEl.classList.add("hidden");
    containerEl.classList.remove("hidden");

    const maxTotal = Math.max(...results.map((r) => r.totalPts || 0), 1);

    let html = "";
    results.forEach(function (entry) {
        const rank      = entry.rank;
        const publicPts = entry.escPublicPts || 0;
        const juryPts   = entry.juryPts || 0;
        const totalPts  = entry.totalPts || 0;
        const flag      = getFlagUrl(entry.countryId);
        const flagUrl   = flag?.url ?? null;
        const changed   = previousData
            ? (previousData.find((p) => p.id === entry.id) || {}).totalVotes !== totalPts
            : false;

        const totalBadgeCls =
            rank === 1
                ? "bg-yellow-400/20 text-yellow-300 border-yellow-400/30"
                : rank <= 3
                  ? "bg-sky-400/15 text-sky-300 border-sky-400/30"
                  : "bg-esc-surface/50 text-esc-muted border-esc-surface";

        html += `
        <div class="group flex items-center gap-3 rounded-xl border border-esc-surface/60 bg-esc-surface/30 px-4 py-3 transition-all duration-500 hover:border-esc-surface ${changed ? "ring-1 ring-esc-yellow/30 bg-esc-yellow/5" : ""}">

            <!-- Rank -->
            <div class="flex-shrink-0 w-8 text-right">
                <span class="font-heading font-bold text-sm ${rank <= 3 ? "text-esc-yellow" : "text-esc-muted"}">
                    ${rank === 1 ? "🥇" : rank === 2 ? "🥈" : rank === 3 ? "🥉" : "#" + rank}
                </span>
            </div>

            <!-- Flag -->
            <div class="flex-shrink-0 w-8 h-8 rounded-full overflow-hidden border border-esc-surface flex items-center justify-center bg-esc-black">
                ${
                    flagUrl
                        ? `<img src="${flagUrl}" alt="${escHtml(entry.country)}" class="w-full h-full object-cover" onerror="this.onerror=null;this.parentElement.innerHTML='<span class=\\'text-sm\\'>🏳️</span>';">`
                        : '<span class="text-sm">🏳️</span>'
                }
            </div>

            <!-- Country + Song -->
            <div class="flex-shrink-0 w-32 sm:w-44 min-w-0">
                <p class="font-heading font-semibold text-sm text-esc-text truncate leading-tight">${escHtml(entry.country)}</p>
                <p class="text-[10px] text-esc-muted truncate leading-tight">${escHtml(entry.name)}</p>
            </div>

            <!-- Bar (proportional to total points) -->
            <div class="flex-1 min-w-0">
                <div class="flex h-6 rounded-md overflow-hidden bg-esc-black/50">
                    <!-- Jury portion -->
                    <div class="h-full transition-all duration-700 ease-out bg-sky-400/70"
                         style="width:${(juryPts / maxTotal) * 100}%;"
                         title="${juryPts} jury pts"></div>
                    <!-- Public portion -->
                    <div class="h-full transition-all duration-700 ease-out bg-esc-yellow/70"
                         style="width:${(publicPts / maxTotal) * 100}%;"
                         title="${publicPts} public pts"></div>
                </div>
            </div>

            <!-- Score badges -->
            <div class="flex-shrink-0 text-right flex items-center gap-1.5">
                <span class="inline-flex items-center rounded-md border border-sky-400/30 bg-sky-400/15 px-1.5 py-0.5 font-heading font-bold text-xs text-sky-300" title="Jury points">
                    J ${juryPts}
                </span>
                <span class="inline-flex items-center rounded-md border border-esc-yellow/30 bg-esc-yellow/10 px-1.5 py-0.5 font-heading font-bold text-xs text-esc-yellow" title="Public vote points">
                    P ${publicPts}
                </span>
                <span class="inline-flex items-center rounded-md border px-2 py-0.5 font-heading font-bold text-sm ${totalBadgeCls}" title="Total points">
                    ${totalPts}
                </span>
            </div>
        </div>`;
    });

    document.getElementById("results-rows").innerHTML = html;
    previousData = results;
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------
async function fetchResults() {
    try {
        const resp = await fetch("/api/results");
        if (!resp.ok) throw new Error("HTTP " + resp.status);
        const data = await resp.json();
        renderResults(data);
    } catch (err) {
        console.error("Failed to fetch results:", err);
        if (!previousData) {
            document.getElementById("loading-state").classList.add("hidden");
            document.getElementById("error-state").classList.remove("hidden");
        }
    }
}

// ---------------------------------------------------------------------------
// Polling + pause
// ---------------------------------------------------------------------------
function resetCountdown() {
    secondsLeft = 10;
    const el = document.getElementById("countdown");
    if (el) el.textContent = secondsLeft;
}

function startPolling() {
    fetchResults();
    countdownInterval = setInterval(function () {
        if (paused) return;
        secondsLeft--;
        if (secondsLeft < 0) secondsLeft = 0;
        const el = document.getElementById("countdown");
        if (el) el.textContent = secondsLeft;
    }, 1000);
    pollInterval = setInterval(function () {
        if (paused) return;
        fetchResults();
        resetCountdown();
    }, 10000);
}

// ---------------------------------------------------------------------------
// Pause / resume
// ---------------------------------------------------------------------------
function togglePause() {
    paused = !paused;
    const btn      = document.getElementById("pause-btn");
    const pingDot  = document.getElementById("ping-dot");
    const refreshT = document.getElementById("refresh-text");
    if (paused) {
        btn.textContent = "Resume";
        pingDot.classList.remove("animate-ping", "bg-green-400");
        pingDot.classList.add("bg-esc-muted");
        refreshT.innerHTML = "Paused";
    } else {
        btn.textContent = "Pause";
        pingDot.classList.add("animate-ping", "bg-green-400");
        pingDot.classList.remove("bg-esc-muted");
        refreshT.innerHTML = 'Live — updates in <span id="countdown">10</span>s';
        resetCountdown();
        fetchResults();
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function escHtml(str) {
    if (!str) return "";
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

// ---------------------------------------------------------------------------
// Boot
// ---------------------------------------------------------------------------
startPolling();
