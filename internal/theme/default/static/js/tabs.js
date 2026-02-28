/**
 * OSG Tabs — zero-dependency tab switching for the tabs shortcode.
 *
 * For each .tabs container, reads data-tab-title from child .tab elements,
 * builds a .tabs-nav bar with .tab-btn buttons, and toggles visibility
 * via the .active class. First tab is active by default.
 */
(function () {
  "use strict";

  var containers = document.querySelectorAll(".tabs");
  if (!containers.length) return;

  containers.forEach(function (container) {
    var tabs = container.querySelectorAll(":scope > .tab");
    if (!tabs.length) return;

    // Build navigation bar.
    var nav = document.createElement("div");
    nav.className = "tabs-nav";
    nav.setAttribute("role", "tablist");

    tabs.forEach(function (tab, index) {
      var title = tab.getAttribute("data-tab-title") || "Tab";

      var btn = document.createElement("button");
      btn.className = "tab-btn";
      btn.textContent = title;
      btn.setAttribute("role", "tab");
      btn.setAttribute("aria-selected", index === 0 ? "true" : "false");
      btn.setAttribute("tabindex", index === 0 ? "0" : "-1");

      btn.addEventListener("click", function () {
        activate(container, index);
      });

      nav.appendChild(btn);
    });

    container.insertBefore(nav, container.firstChild);

    // Activate first tab.
    activate(container, 0);
  });

  function activate(container, index) {
    var tabs = container.querySelectorAll(":scope > .tab");
    var btns = container.querySelectorAll(".tabs-nav .tab-btn");

    tabs.forEach(function (tab, i) {
      tab.classList.toggle("active", i === index);
    });

    btns.forEach(function (btn, i) {
      btn.classList.toggle("active", i === index);
      btn.setAttribute("aria-selected", i === index ? "true" : "false");
      btn.setAttribute("tabindex", i === index ? "0" : "-1");
    });
  }

  // Keyboard navigation within tab bar.
  document.addEventListener("keydown", function (e) {
    var btn = document.activeElement;
    if (!btn || !btn.classList.contains("tab-btn")) return;

    var nav = btn.parentElement;
    var btns = Array.prototype.slice.call(nav.querySelectorAll(".tab-btn"));
    var idx = btns.indexOf(btn);
    var newIdx = -1;

    if (e.key === "ArrowRight" || e.key === "ArrowDown") {
      newIdx = (idx + 1) % btns.length;
    } else if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
      newIdx = (idx - 1 + btns.length) % btns.length;
    } else if (e.key === "Home") {
      newIdx = 0;
    } else if (e.key === "End") {
      newIdx = btns.length - 1;
    }

    if (newIdx >= 0) {
      e.preventDefault();
      btns[newIdx].focus();
      btns[newIdx].click();
    }
  });
})();
