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
    setupRebuildBar();
    setupDrawer();
    setupOperationsPolling();
  });

  // Drawer: opens on htmx:afterSwap when target is #drawer; closes on
  // backdrop click, Esc key, or any element with [data-drawer-close].
  // Tab switching is plain CSS class toggling — no router needed.
  function setupDrawer() {
    const drawer = document.getElementById("drawer");
    const backdrop = document.getElementById("drawer-backdrop");
    if (!drawer || !backdrop) return;

    function open() {
      drawer.classList.add("is-open");
      backdrop.classList.add("is-open");
      drawer.setAttribute("aria-hidden", "false");
      // Default-active tab: first one.
      activateTab(drawer, "output");
      streamDrawerLogsIfNeeded(drawer);
    }
    function close() {
      drawer.classList.remove("is-open");
      backdrop.classList.remove("is-open");
      drawer.setAttribute("aria-hidden", "true");
      stopDrawerStream();
    }

    document.body.addEventListener("htmx:afterSwap", function (e) {
      if (e.detail && e.detail.target && e.detail.target.id === "drawer") {
        open();
      }
    });
    document.addEventListener("click", function (e) {
      const t = e.target;
      if (!t) return;
      if (t.matches && (t.matches("[data-drawer-close]") || t.closest("[data-drawer-close]"))) {
        close();
        return;
      }
      const tab = t.closest && t.closest("[data-drawer-tab]");
      if (tab && drawer.contains(tab)) {
        activateTab(drawer, tab.getAttribute("data-drawer-tab"));
      }
    });
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && drawer.classList.contains("is-open")) close();
    });
  }

  function activateTab(drawer, name) {
    drawer.querySelectorAll("[data-drawer-tab]").forEach(function (b) {
      b.classList.toggle("is-active", b.getAttribute("data-drawer-tab") === name);
    });
    drawer.querySelectorAll("[data-drawer-pane]").forEach(function (p) {
      p.classList.toggle("is-active", p.getAttribute("data-drawer-pane") === name);
    });
  }

  let drawerEventSource = null;
  function streamDrawerLogsIfNeeded(drawer) {
    stopDrawerStream();
    const pre = drawer.querySelector("[data-drawer-logs]");
    if (!pre) return;
    const name = pre.getAttribute("data-drawer-logs");
    if (!name) return;
    drawerEventSource = new EventSource("/operations/" + encodeURIComponent(name) + "/logs");
    drawerEventSource.addEventListener("log", function (ev) {
      const nearBottom = pre.scrollTop + pre.clientHeight >= pre.scrollHeight - 20;
      pre.textContent += ev.data + "\n";
      if (nearBottom) pre.scrollTop = pre.scrollHeight;
    });
    drawerEventSource.onerror = function () {
      // Browser will retry; if it doesn't reconnect that's fine — drawer
      // shows the buffered tail until the user closes it.
    };
  }
  function stopDrawerStream() {
    if (drawerEventSource) {
      drawerEventSource.close();
      drawerEventSource = null;
    }
  }

  // Polls /operations.json for the /actions page so cards keep their
  // pill state and uptime in sync without a full reload.
  function setupOperationsPolling() {
    const root = document.querySelector("[data-operations-poll]");
    if (!root) return;

    const pillClass = {
      running: "status-pill is-running",
      starting: "status-pill is-running",
      idle: "status-pill is-idle",
      error: "status-pill is-error",
      cancelled: "status-pill is-warn",
      stopping: "status-pill is-warn",
    };

    function update() {
      fetch("/operations.json", { headers: { Accept: "application/json" } })
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          if (!data || !Array.isArray(data.operations)) return;
          data.operations.forEach(function (op) {
            const card = root.querySelector('[data-operation="' + cssEscape(op.name) + '"]');
            if (!card) return;
            const pill = card.querySelector("[data-state-pill]");
            if (pill) {
              pill.textContent = op.state;
              pill.className = pillClass[op.state] || "status-pill is-idle";
            }
          });
        })
        .catch(function () {});
    }
    function cssEscape(s) {
      if (window.CSS && CSS.escape) return CSS.escape(s);
      return String(s).replace(/["\\]/g, "\\$&");
    }
    update();
    setInterval(update, 2500);
  }

  function setupRebuildBar() {
    const bar = document.querySelector("[data-rebuild-bar]");
    if (!bar) return;
    const button = bar.querySelector("[data-rebuild-button]");
    const status = bar.querySelector("[data-rebuild-status]");

    function update() {
      fetch("/rebuild.json", { headers: { Accept: "application/json" } })
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          if (!data) return;
          if (data.running) {
            if (button) {
              button.disabled = true;
              button.textContent = "Rebuilding…";
            }
            if (status) status.textContent = "in progress";
          } else {
            if (button) {
              button.disabled = false;
              button.textContent = "Rebuild now";
            }
            if (status && data.last_ran) {
              const t = new Date(data.last_ran).toLocaleTimeString();
              const ms = data.duration_ms || 0;
              const err = data.last_error ? " — error" : "";
              status.textContent = "last ran " + t + " (" + ms + "ms)" + err;
            }
          }
        })
        .catch(function () {});
    }

    setInterval(update, 1500);
  }

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

    // Read the noun ("pages", "files"…) from the count element's initial
    // text so each page can label its rows appropriately. Falls back to
    // "items" if no initial text is present.
    const initial = (count && count.textContent.trim()) || "";
    const noun = initial.split(/\s+/).slice(1).join(" ") || "items";

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
          ? total + " " + noun
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
