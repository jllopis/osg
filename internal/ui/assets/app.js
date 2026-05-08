// OSG UI — minimal client-side helpers.
(function () {
  "use strict";

  const STORAGE_KEY = "osg-ui-theme"; // "light" | "dark" | null (auto)
  const root = document.documentElement;

  function apply(theme) {
    root.classList.remove("light", "dark");
    if (theme === "light" || theme === "dark") {
      root.classList.add(theme);
    }
  }

  function current() {
    return localStorage.getItem(STORAGE_KEY);
  }

  function next(theme) {
    if (theme === null) return "dark";
    if (theme === "dark") return "light";
    return null; // auto
  }

  function label(theme) {
    if (theme === "light") return "Light";
    if (theme === "dark") return "Dark";
    return "Auto";
  }

  apply(current());

  document.addEventListener("DOMContentLoaded", function () {
    const btn = document.querySelector("[data-theme-toggle]");
    if (btn) {
      btn.textContent = label(current());
      btn.addEventListener("click", function () {
        const n = next(current());
        if (n === null) {
          localStorage.removeItem(STORAGE_KEY);
        } else {
          localStorage.setItem(STORAGE_KEY, n);
        }
        apply(n);
        btn.textContent = label(n);
      });
    }

    setupServiceLogs();
    setupServicesPolling();
    setupVaultFilter();
  });

  function setupVaultFilter() {
    const input = document.querySelector("[data-vault-filter]");
    const table = document.querySelector("[data-vault-table]");
    const count = document.querySelector("[data-vault-count]");
    const empty = document.querySelector("[data-vault-empty]");
    if (!input || !table) return;

    const rows = Array.from(table.querySelectorAll("tbody tr"));
    const total = rows.length;
    const haystacks = rows.map(function (r) {
      return (r.getAttribute("data-search") || r.textContent || "").toLowerCase();
    });

    function apply() {
      const q = input.value.trim().toLowerCase();
      let shown = 0;
      rows.forEach(function (r, i) {
        const match = q === "" || haystacks[i].indexOf(q) !== -1;
        r.hidden = !match;
        if (match) shown++;
      });
      if (count) {
        count.textContent = q === ""
          ? total + " pages"
          : shown + " of " + total;
      }
      if (empty) empty.hidden = shown !== 0;
    }

    input.addEventListener("input", apply);
    apply();
  }

  function setupServicesPolling() {
    const root = document.querySelector("[data-services-poll]");
    if (!root) return;

    const pillClass = {
      running: "pill ok",
      starting: "pill info",
      stopping: "pill warn",
      error: "pill err",
      idle: "pill muted",
    };

    function update() {
      fetch("/services.json", { headers: { Accept: "application/json" } })
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          if (!data || !Array.isArray(data.services)) return;
          data.services.forEach(applyToCard);
        })
        .catch(function () {});
    }

    function applyToCard(svc) {
      const card = root.querySelector('[data-service="' + cssEscape(svc.name) + '"]');
      if (!card) return;

      const pill = card.querySelector("[data-state-pill]");
      if (pill) {
        pill.textContent = svc.state;
        pill.className = pillClass[svc.state] || "pill muted";
      }

      const running = svc.state === "running" || svc.state === "starting";
      card.querySelectorAll("[data-uptime-row]").forEach(function (el) {
        el.hidden = !running;
      });
      const uptime = card.querySelector("[data-uptime]");
      if (uptime && running && svc.uptime) uptime.textContent = svc.uptime;

      card.querySelectorAll("[data-error-row]").forEach(function (el) {
        el.hidden = !svc.last_error;
      });
      const errEl = card.querySelector("[data-error]");
      if (errEl && svc.last_error) errEl.textContent = svc.last_error;

      const form = card.querySelector("[data-action-form]");
      const btn = card.querySelector("[data-action-button]");
      if (form && btn) {
        if (running) {
          form.setAttribute("action", "/services/stop");
          btn.textContent = "Stop";
          btn.className = "btn btn-stop";
        } else {
          form.setAttribute("action", "/services/start");
          btn.textContent = "Start";
          btn.className = "btn btn-start";
        }
      }
    }

    function cssEscape(s) {
      if (window.CSS && CSS.escape) return CSS.escape(s);
      return String(s).replace(/["\\]/g, "\\$&");
    }

    update();
    setInterval(update, 2000);
  }

  function setupServiceLogs() {
    const blocks = document.querySelectorAll("[data-service-logs]");
    blocks.forEach(function (det) {
      const name = det.getAttribute("data-service-logs");
      const pre = det.querySelector("pre.logs");
      if (!name || !pre) return;
      let es = null;

      function start() {
        if (es) return;
        es = new EventSource("/services/" + encodeURIComponent(name) + "/logs");
        es.addEventListener("log", function (ev) {
          const line = ev.data;
          // Auto-scroll only if the user is already near the bottom.
          const nearBottom = pre.scrollTop + pre.clientHeight >= pre.scrollHeight - 20;
          pre.textContent += line + "\n";
          // Trim very long buffers in the DOM (~2000 lines).
          const lines = pre.textContent.split("\n");
          if (lines.length > 2000) {
            pre.textContent = lines.slice(lines.length - 2000).join("\n");
          }
          if (nearBottom) pre.scrollTop = pre.scrollHeight;
        });
        es.onerror = function () {
          // Browser will auto-reconnect; if persistent, close it.
        };
      }
      function stop() {
        if (es) {
          es.close();
          es = null;
        }
      }

      // Replace pre content with the initial server-rendered content
      // when streaming starts (so we don't double-show history that the
      // server side included as ringBuffer.Tail). We let the SSE replay
      // history first and use that as the source of truth.
      det.addEventListener("toggle", function () {
        if (det.open) {
          pre.textContent = "";
          start();
        } else {
          stop();
        }
      });

      // If the panel is open on first paint (e.g. user navigates back),
      // start immediately.
      if (det.open) {
        pre.textContent = "";
        start();
      }
    });
  }
})();
