/**
 * OSG Interactions — Page views and likes/dislikes.
 *
 * Loaded only when config.interactions_enabled is true.
 * Generates a client-side fingerprint using a UUID stored in localStorage
 * combined with browser characteristics, hashed with SHA-256.
 *
 * No IP addresses are used (unreliable behind proxies / dynamic IPs).
 */
(function () {
  "use strict";

  var container = document.getElementById("osg-interactions");
  if (!container) return;

  var apiURL = container.dataset.apiUrl || "";
  var pagePath = container.dataset.pagePath || window.location.pathname;

  // --- Fingerprinting ---

  /**
   * Returns a stable fingerprint string based on a UUID stored in
   * localStorage + browser characteristics, hashed with SHA-256.
   */
  async function getFingerprint() {
    var key = "osg-fp-uuid";
    var uuid = localStorage.getItem(key);
    if (!uuid) {
      uuid = crypto.randomUUID
        ? crypto.randomUUID()
        : "xxxx-xxxx-xxxx".replace(/x/g, function () {
            return ((Math.random() * 16) | 0).toString(16);
          });
      localStorage.setItem(key, uuid);
    }

    var raw = [
      uuid,
      navigator.userAgent || "",
      screen.width + "x" + screen.height,
      window.devicePixelRatio || 1,
      Intl.DateTimeFormat().resolvedOptions().timeZone || "",
      navigator.language || "",
      navigator.platform || "",
      navigator.hardwareConcurrency || 0,
      screen.colorDepth || 0,
    ].join("|");

    var buf = new TextEncoder().encode(raw);
    var hash = await crypto.subtle.digest("SHA-256", buf);
    var arr = Array.from(new Uint8Array(hash));
    return arr.map(function (b) { return b.toString(16).padStart(2, "0"); }).join("");
  }

  // --- API helpers ---

  function post(endpoint, body) {
    return fetch(apiURL + endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    });
  }

  // --- UI rendering ---

  function formatNumber(n) {
    if (n >= 1000000) return (n / 1000000).toFixed(1) + "M";
    if (n >= 1000) return (n / 1000).toFixed(1) + "K";
    return String(n);
  }

  function updateUI(stats) {
    var views = container.querySelector(".interactions-views-count");
    var likes = container.querySelector(".interactions-likes-count");
    var dislikes = container.querySelector(".interactions-dislikes-count");
    var likeBtn = container.querySelector(".interactions-like-btn");
    var dislikeBtn = container.querySelector(".interactions-dislike-btn");

    if (views) views.textContent = formatNumber(stats.views);
    if (likes) likes.textContent = formatNumber(stats.likes);
    if (dislikes) dislikes.textContent = formatNumber(stats.dislikes);

    // Active state.
    if (likeBtn) {
      likeBtn.classList.toggle("active", stats.user_vote === 1);
      likeBtn.setAttribute("aria-pressed", stats.user_vote === 1);
    }
    if (dislikeBtn) {
      dislikeBtn.classList.toggle("active", stats.user_vote === -1);
      dislikeBtn.setAttribute("aria-pressed", stats.user_vote === -1);
    }
  }

  // --- Main ---

  async function init() {
    var fp;
    try {
      fp = await getFingerprint();
    } catch (_) {
      // Fingerprinting not available (e.g. insecure context). Silently skip.
      return;
    }

    // Record view and get initial stats.
    try {
      var stats = await post("/api/v1/pageview", { path: pagePath, fp: fp });
      updateUI(stats);
    } catch (_) {
      // API unavailable — leave UI in default state.
    }

    // Like button.
    var likeBtn = container.querySelector(".interactions-like-btn");
    if (likeBtn) {
      likeBtn.addEventListener("click", async function () {
        var currentVote = likeBtn.classList.contains("active") ? 1 : 0;
        var newVote = currentVote === 1 ? 0 : 1;
        try {
          var s = await post("/api/v1/vote", { path: pagePath, fp: fp, vote: newVote });
          updateUI(s);
        } catch (_) { /* ignore */ }
      });
    }

    // Dislike button.
    var dislikeBtn = container.querySelector(".interactions-dislike-btn");
    if (dislikeBtn) {
      dislikeBtn.addEventListener("click", async function () {
        var currentVote = dislikeBtn.classList.contains("active") ? -1 : 0;
        var newVote = currentVote === -1 ? 0 : -1;
        try {
          var s = await post("/api/v1/vote", { path: pagePath, fp: fp, vote: newVote });
          updateUI(s);
        } catch (_) { /* ignore */ }
      });
    }
  }

  init();
})();
