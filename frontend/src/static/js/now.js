// -----------------------------------------------------------------------
// Constants (SONG_ID, SERVER_RUN_ID, CONTEST_ACTIVE, TOTAL_POINTS,
// localRemaining, currentPoints) are injected by the template as
// globals before this script is loaded.
// -----------------------------------------------------------------------

// Set the flag for the current entry
(function() {
    const el = document.getElementById("flag-display");
    if (el) {
        const code = el.dataset.countryId;
        el.textContent = code;
        el.previousElementSibling && (el.previousElementSibling.style.display = "none");
    }
})();

// ---------------------------------------------------------------------------
// Point stepper
// ---------------------------------------------------------------------------
function stepPoints(delta) {
    const newVal = currentPoints + delta;
    if (newVal < 1 || newVal > localRemaining) return;
    currentPoints = newVal;
    updateStepperUI();
}

function updateStepperUI() {
    const disp  = document.getElementById("points-display");
    const minus = document.getElementById("minus-btn");
    const plus  = document.getElementById("plus-btn");
    if (disp)  disp.textContent = currentPoints;
    if (minus) minus.disabled   = currentPoints <= 1;
    if (plus)  plus.disabled    = currentPoints >= localRemaining;
}

function refreshBudgetUI() {
    const bar   = document.getElementById("budget-bar");
    const label = document.getElementById("budget-display");
    if (bar)   bar.style.width = Math.max(0, (localRemaining / TOTAL_POINTS) * 100) + "%";
    if (label) label.textContent = localRemaining;
    updateStepperUI();

    const voteBtn = document.getElementById("vote-btn");
    if (voteBtn) voteBtn.disabled = localRemaining <= 0;
}

// ---------------------------------------------------------------------------
// Vote submission
// ---------------------------------------------------------------------------
async function submitVote() {
    if (typeof window.hasVoteCookieConsent === "function" && !window.hasVoteCookieConsent()) {
        showVoteResponse("error", "Please accept required vote cookies before submitting votes.");
        if (typeof window.revealCookieBanner === "function") {
            window.revealCookieBanner();
        }
        return;
    }

    const phone      = (document.getElementById("phone-input")?.value || "").trim();
    const ownCountry = (document.getElementById("own-country-select")?.value || "");
    const btn        = document.getElementById("vote-btn");
    const btnText    = document.getElementById("vote-btn-text");
    const spinner    = document.getElementById("vote-spinner");

    if (!phone)      { showVoteResponse("error", "Please enter your phone number."); return; }
    if (!ownCountry) { showVoteResponse("error", "Please select your country."); return; }
    if (currentPoints < 1)    { showVoteResponse("error", "Select at least 1 point."); return; }
    if (localRemaining <= 0)  { showVoteResponse("error", "No vote points remaining."); return; }

    if (btn)     btn.disabled = true;
    if (btnText) btnText.textContent = "Submitting…";
    if (spinner) spinner.classList.remove("hidden");
    document.getElementById("vote-response")?.classList.add("hidden");

    const formData = new FormData();
    formData.append("songID",     SONG_ID);
    formData.append("phoneNum",   phone);
    formData.append("ownCountry", ownCountry);
    formData.append("points",     currentPoints);

    try {
        const resp = await fetch("/vote/submit", { method: "POST", body: formData });
        const data = await resp.json();

        if (resp.ok) {
            const newRemaining = typeof data.votes_remaining !== "undefined"
                ? data.votes_remaining
                : localRemaining - currentPoints;
            localRemaining = Math.max(0, newRemaining);
            currentPoints  = Math.min(currentPoints, Math.max(1, localRemaining));
            refreshBudgetUI();

            if (typeof data.totalVotes !== "undefined") {
                const sc = document.getElementById("live-score");
                if (sc) sc.textContent = data.totalVotes;
            }

            showVoteResponse("success", `✓ Vote submitted! ${localRemaining} point${localRemaining !== 1 ? "s" : ""} remaining.`);
        } else {
            showVoteResponse("error", data.error || data.message || "Failed to submit vote.");
        }
    } catch (e) {
        showVoteResponse("error", "Network error — please try again.");
    } finally {
        if (btnText) btnText.textContent = "Submit Vote";
        if (spinner) spinner.classList.add("hidden");
        if (btn && localRemaining > 0) btn.disabled = false;
    }
}

function showVoteResponse(type, msg) {
    const el = document.getElementById("vote-response");
    if (!el) return;
    el.classList.remove("hidden",
        "border-green-500/30", "bg-green-500/10", "text-green-400",
        "border-red-500/30",   "bg-red-500/10",   "text-red-400",
        "border-esc-yellow/30","bg-esc-yellow/10","text-esc-yellow");
    const styles = {
        success: ["border-green-500/30", "bg-green-500/10", "text-green-400"],
        error:   ["border-red-500/30",   "bg-red-500/10",   "text-red-400"],
        info:    ["border-esc-yellow/30","bg-esc-yellow/10","text-esc-yellow"],
    };
    el.classList.add("border", ...(styles[type] || styles.info));
    el.textContent = msg;
    el.classList.remove("hidden");
}

// ---------------------------------------------------------------------------
// Auto-poll: detect when the admin has moved to the next song and reload.
// Called from the template when contest_active and song are set.
// ---------------------------------------------------------------------------
function initPoll(runId, startIndex) {
    let _pollRunId  = runId;
    let _pollIndex  = startIndex;
    let _interval   = null;

    async function pollContestState() {
        try {
            const resp = await fetch("/api/contest/current");
            if (!resp.ok) return;
            const data    = await resp.json();
            const payload = data.payload;
            if (!payload) {
                if (resp.status === 404 || data.error) {
                    clearInterval(_interval);
                    showContestEndedBanner();
                }
                return;
            }
            if (payload.runId !== _pollRunId || payload.currentIndex !== _pollIndex) {
                clearInterval(_interval);
                window.location.reload();
            }
            const sc = document.getElementById("live-score");
            if (sc && typeof payload.totalVotes !== "undefined") {
                sc.textContent = payload.totalVotes;
            }
        } catch (e) { /* silent */ }
    }

    _interval = setInterval(pollContestState, 5000);
}

function showContestEndedBanner() {
    const banner = document.createElement("div");
    banner.className = "fixed bottom-6 left-1/2 -translate-x-1/2 z-50 rounded-2xl border border-esc-yellow/40 bg-esc-black shadow-2xl px-6 py-4 text-center max-w-sm";
    banner.innerHTML = `
        <p class="font-heading font-bold text-esc-yellow text-lg mb-1">🏆 Contest Finished!</p>
        <p class="text-xs text-esc-muted mb-3">All songs have performed. Check the results!</p>
        <a href="/results" class="inline-flex items-center gap-1.5 rounded-lg bg-esc-yellow px-4 py-2 text-xs font-heading font-bold text-esc-black uppercase tracking-wider hover:bg-yellow-400 transition-colors">
            View Results
        </a>`;
    document.body.appendChild(banner);
}

// ---------------------------------------------------------------------------
// Keyboard shortcut: +/- for point stepper
// ---------------------------------------------------------------------------
document.addEventListener("keydown", e => {
    if (e.target.tagName === "INPUT" || e.target.tagName === "SELECT") return;
    if (e.key === "+" || e.key === "=") stepPoints(1);
    if (e.key === "-")                  stepPoints(-1);
    if (e.key === "Enter" && CONTEST_ACTIVE) submitVote();
});
