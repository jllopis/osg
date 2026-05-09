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
    setupConfirmDialogs();
  });

  // setupConfirmDialogs intercepts submit on any form with [data-confirm]
  // and shows a native <dialog> modal. Submission is blocked until the
  // user clicks "Continue" — Cancel just closes the dialog.
  function setupConfirmDialogs() {
    let dialog = document.getElementById("confirm-dialog");
    if (!dialog) {
      dialog = document.createElement("dialog");
      dialog.id = "confirm-dialog";
      dialog.className = "confirm-dialog";
      dialog.innerHTML =
        '<h3>Confirm action</h3>' +
        '<p data-confirm-text></p>' +
        '<div class="actions">' +
        '<button type="button" class="btn" data-confirm-cancel>Cancel</button>' +
        '<button type="button" class="btn btn-run" data-confirm-ok>Continue</button>' +
        '</div>';
      document.body.appendChild(dialog);
    }
    const text = dialog.querySelector("[data-confirm-text]");
    const cancelBtn = dialog.querySelector("[data-confirm-cancel]");
    const okBtn = dialog.querySelector("[data-confirm-ok]");
    let pendingForm = null;

    cancelBtn.addEventListener("click", function () { dialog.close(); pendingForm = null; });
    okBtn.addEventListener("click", function () {
      const f = pendingForm;
      dialog.close();
      pendingForm = null;
      if (f) {
        // Bypass interception for the second submit.
        f.dataset.confirmed = "1";
        f.submit();
      }
    });

    document.body.addEventListener("submit", function (e) {
      const form = e.target;
      if (!form || !form.hasAttribute("data-confirm")) return;
      if (form.dataset.confirmed === "1") return;
      e.preventDefault();
      text.textContent = form.getAttribute("data-confirm");
      pendingForm = form;
      if (typeof dialog.showModal === "function") {
        dialog.showModal();
      } else {
        // Older browsers: fall back to confirm().
        if (window.confirm(form.getAttribute("data-confirm"))) {
          form.dataset.confirmed = "1";
          form.submit();
        }
        pendingForm = null;
      }
    });
  }

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

  // setupFlowDrawer wires the bottom panel on /actions that follows the
  // currently running pipeline step. The panel auto-opens when a step
  // enters running/starting state, attaches an EventSource to its log
  // stream, and switches its source as each step finishes and the next
  // takes over. The user can resize the panel by dragging the top edge
  // and clear the buffer or close the whole panel via the header
  // controls. Returns null when the page has no flow drawer (i.e.
  // anywhere other than /actions).
  function setupFlowDrawer() {
    const drawer = document.querySelector("[data-flow-drawer]");
    if (!drawer) return null;
    const stepEl = drawer.querySelector("[data-flow-step]");
    const statusEl = drawer.querySelector("[data-flow-status]");
    const logsEl = drawer.querySelector("[data-flow-logs]");
    const closeBtn = drawer.querySelector("[data-flow-close]");
    const clearBtn = drawer.querySelector("[data-flow-clear]");
    const resizer = drawer.querySelector("[data-flow-resize]");

    let following = null;
    let es = null;

    function openIfHidden() { if (drawer.hidden) drawer.hidden = false; }
    function closeStream() {
      if (es) { es.close(); es = null; }
    }
    function appendLine(line) {
      const nearBottom = logsEl.scrollTop + logsEl.clientHeight >= logsEl.scrollHeight - 20;
      logsEl.appendChild(document.createTextNode(line + "\n"));
      if (nearBottom) logsEl.scrollTop = logsEl.scrollHeight;
    }
    function appendSeparator(text) {
      const span = document.createElement("span");
      span.className = "flow-drawer-sep";
      span.textContent = "── " + text + " ──";
      logsEl.appendChild(span);
      logsEl.appendChild(document.createTextNode("\n"));
      logsEl.scrollTop = logsEl.scrollHeight;
    }
    function follow(name) {
      closeStream();
      following = name;
      stepEl.textContent = name;
      statusEl.textContent = "running…";
      appendSeparator(name + " started");
      es = new EventSource("/operations/" + encodeURIComponent(name) + "/logs");
      es.addEventListener("log", function (ev) { appendLine(ev.data); });
      es.onerror = function () { /* run end closes the stream; pill update reports final state */ };
    }
    function unfollow(prevState) {
      if (!following) return;
      appendSeparator(following + " " + (prevState || "done"));
      closeStream();
      stepEl.textContent = "—";
      statusEl.textContent = "idle";
      following = null;
    }

    closeBtn.addEventListener("click", function () {
      closeStream();
      following = null;
      drawer.hidden = true;
    });
    clearBtn.addEventListener("click", function () {
      logsEl.textContent = "";
    });

    // Resize: drag the top edge, persist height to localStorage so it
    // survives page navigation. Lower bound matches CSS min-height; the
    // upper bound leaves space for the pipeline above.
    let dragging = false, startY = 0, startH = 0;
    resizer.addEventListener("mousedown", function (e) {
      dragging = true; startY = e.clientY; startH = drawer.offsetHeight;
      drawer.classList.add("is-resizing");
      e.preventDefault();
    });
    document.addEventListener("mousemove", function (e) {
      if (!dragging) return;
      const dy = startY - e.clientY;
      const h = Math.max(140, Math.min(window.innerHeight - 120, startH + dy));
      drawer.style.height = h + "px";
    });
    document.addEventListener("mouseup", function () {
      if (!dragging) return;
      dragging = false;
      drawer.classList.remove("is-resizing");
      try { localStorage.setItem("osg-flow-drawer-h", String(drawer.offsetHeight)); } catch (_) {}
    });
    try {
      const saved = parseInt(localStorage.getItem("osg-flow-drawer-h") || "0", 10);
      if (saved > 140) drawer.style.height = saved + "px";
    } catch (_) {}

    return {
      // update is called on every operations poll cycle. ops is the
      // map name → op JSON; running is the first op found in a
      // running/starting state (or null when the pipeline is idle).
      update: function (running, ops) {
        if (running) {
          if (running.name !== following) {
            if (following) {
              const prev = ops.get(following);
              appendSeparator(following + " " + (prev ? prev.state : "done"));
            }
            openIfHidden();
            follow(running.name);
          }
        } else if (following) {
          const prev = ops.get(following);
          unfollow(prev ? prev.state : "done");
        }
      },
    };
  }

  // Polls /operations.json so cards (flow nodes, op-cards, quick
  // buttons, task-form panels) reflect the runner's current state. The
  // pill text/class is patched in place for cheap feedback; the card is
  // re-fetched and swapped when the state crosses a transition (e.g.
  // running → idle), which also flips the Run/Stop button and meta line.
  function setupOperationsPolling() {
    const cards = document.querySelectorAll("[data-card-style]");
    if (!cards.length) return;

    const pillClass = {
      running: "status-pill is-running",
      starting: "status-pill is-running",
      idle: "status-pill is-idle",
      error: "status-pill is-error",
      cancelled: "status-pill is-warn",
      stopping: "status-pill is-warn",
    };

    const flowDrawer = setupFlowDrawer();
    const inFlight = new Set();

    function swapCard(card, op) {
      const style = card.getAttribute("data-card-style");
      if (!style) return;
      const key = op.name + "/" + style;
      if (inFlight.has(key)) return;
      inFlight.add(key);
      fetch("/operations/" + encodeURIComponent(op.name) + "/card?style=" + encodeURIComponent(style))
        .then(function (r) { return r.ok ? r.text() : null; })
        .then(function (html) {
          if (!html) return;
          const tmpl = document.createElement("template");
          tmpl.innerHTML = html.trim();
          const next = tmpl.content.firstElementChild;
          if (next && card.parentNode) {
            card.replaceWith(next);
          }
        })
        .catch(function () {})
        .finally(function () { inFlight.delete(key); });
    }

    function update() {
      fetch("/operations.json", { headers: { Accept: "application/json" } })
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          if (!data || !Array.isArray(data.operations)) return;
          const byName = new Map(data.operations.map(function (op) { return [op.name, op]; }));
          let running = null;
          data.operations.forEach(function (op) {
            if (!running && (op.state === "running" || op.state === "starting")) {
              running = op;
            }
          });
          if (flowDrawer) flowDrawer.update(running, byName);
          document.querySelectorAll("[data-card-style]").forEach(function (card) {
            const name = card.getAttribute("data-operation");
            const op = name ? byName.get(name) : null;
            if (!op) return;
            const prev = card.getAttribute("data-state");
            if (prev !== op.state) {
              swapCard(card, op);
              return;
            }
            const pill = card.querySelector("[data-state-pill]");
            if (pill) {
              pill.textContent = op.state;
              pill.className = pillClass[op.state] || "status-pill is-idle";
            }
          });
        })
        .catch(function () {});
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
