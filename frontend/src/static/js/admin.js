// -----------------------------------------------------------------------
// YouTube URL normalizer — converts any YouTube link to embed format
// -----------------------------------------------------------------------
function extractYoutubeId(url) {
    if (!url) return null;
    const patterns = [
        /youtube\.com\/embed\/([A-Za-z0-9_-]{11})/,
        /youtu\.be\/([A-Za-z0-9_-]{11})/,
        /youtube\.com\/watch\?(?:.*&)?v=([A-Za-z0-9_-]{11})/,
        /youtube\.com\/shorts\/([A-Za-z0-9_-]{11})/,
    ];
    for (const re of patterns) {
        const m = url.match(re);
        if (m) return m[1];
    }
    return null;
}

function normalizeYoutubeInput(input) {
    const hint = document.getElementById("song-youtube-url-hint");
    const id   = extractYoutubeId(input.value.trim());
    if (input.value.trim() === "") {
        input.classList.remove("border-green-500/60", "border-red-500/60");
        if (hint)
            hint.textContent =
                "Paste any YouTube link — watch, share, or embed. It will be converted automatically. Shown on the Running Now page.";
        return;
    }
    if (id) {
        input.value = "https://www.youtube.com/embed/" + id;
        input.classList.remove("border-red-500/60");
        input.classList.add("border-green-500/60");
        if (hint) {
            hint.textContent = "✓ Embed URL ready: " + input.value;
            hint.classList.add("text-green-400/70");
            hint.classList.remove("text-esc-muted/60", "text-red-400/70");
        }
    } else {
        input.classList.remove("border-green-500/60");
        input.classList.add("border-red-500/60");
        if (hint) {
            hint.textContent =
                "⚠ Could not recognise a YouTube video ID. Paste a watch, share, or embed link.";
            hint.classList.add("text-red-400/70");
            hint.classList.remove("text-esc-muted/60", "text-green-400/70");
        }
    }
}

let pendingResetForm        = null;
let pendingStartContestForm = null;

function confirmStartContest() {
    const modal = document.getElementById("start-contest-modal");
    modal.classList.remove("hidden");
    document.body.style.overflow = "hidden";

    pendingStartContestForm = event.target.closest("form");

    document.getElementById("confirm-start-contest-btn").onclick = function () {
        const form = pendingStartContestForm;
        cancelStartContest();
        form.submit();
    };

    return false;
}

function cancelStartContest() {
    const modal = document.getElementById("start-contest-modal");
    modal.classList.add("hidden");
    document.body.style.overflow = "";
    pendingStartContestForm = null;
}

function confirmReset() {
    const modal = document.getElementById("reset-modal");
    modal.classList.remove("hidden");
    document.body.style.overflow = "hidden";

    pendingResetForm = event.target.closest("form");

    document.getElementById("confirm-reset-btn").onclick = function () {
        const form = pendingResetForm;
        cancelReset();
        form.submit();
    };

    return false;
}

function cancelReset() {
    const modal = document.getElementById("reset-modal");
    modal.classList.add("hidden");
    document.body.style.overflow = "";
    pendingResetForm = null;
}

document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
        cancelReset();
        cancelStartContest();
    }
});
