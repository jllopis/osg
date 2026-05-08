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
  });

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
