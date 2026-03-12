/**
 * OSG Share — Popover share buttons (X, LinkedIn, Bluesky, Email, Copy link).
 *
 * Works with any number of .share-wrap elements on the page.  Each one
 * contains a toggle button and a .share-popover with the share options.
 */
(function () {
  "use strict";

  // --- Helpers ---

  function resolveURL(path) {
    var a = document.createElement("a");
    a.href = path;
    return a.href;
  }

  function copyToClipboard(text, onSuccess) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(onSuccess).catch(function () {});
    } else {
      var ta = document.createElement("textarea");
      ta.value = text;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand("copy"); onSuccess(); } catch (_) {}
      document.body.removeChild(ta);
    }
  }

  // --- Title permalink (copy on click) ---

  var titleLink = document.querySelector(".title-copy-link");
  if (titleLink) {
    titleLink.addEventListener("click", function (e) {
      e.preventDefault();
      copyToClipboard(titleLink.href, function () {
        titleLink.classList.add("copied");
        setTimeout(function () { titleLink.classList.remove("copied"); }, 2000);
      });
    });
  }

  // --- Share popovers ---

  var wraps = document.querySelectorAll(".share-wrap");
  if (!wraps.length) return;

  // Track the currently open popover so we can close it when another opens.
  var activePopover = null;
  var activeToggle = null;

  function closeActive() {
    if (activePopover) {
      activePopover.hidden = true;
      activeToggle.setAttribute("aria-expanded", "false");
      activePopover = null;
      activeToggle = null;
    }
  }

  wraps.forEach(function (wrap) {
    var toggle = wrap.querySelector(".share-toggle, .share-toggle--compact");
    var popover = wrap.querySelector(".share-popover");
    if (!toggle || !popover) return;

    var permalink = resolveURL(wrap.dataset.permalink || window.location.pathname);
    var title = wrap.dataset.title || document.title;
    var encoded = encodeURIComponent(permalink);
    var encodedTitle = encodeURIComponent(title);

    // Build share URLs.
    var xBtn = popover.querySelector(".share-x");
    if (xBtn) xBtn.href = "https://x.com/intent/tweet?url=" + encoded + "&text=" + encodedTitle;

    var liBtn = popover.querySelector(".share-linkedin");
    if (liBtn) liBtn.href = "https://www.linkedin.com/sharing/share-offsite/?url=" + encoded;

    var bsBtn = popover.querySelector(".share-bluesky");
    if (bsBtn) bsBtn.href = "https://bsky.app/intent/compose?text=" + encodedTitle + " " + encoded;

    var emailBtn = popover.querySelector(".share-email");
    if (emailBtn) emailBtn.href = "mailto:?subject=" + encodedTitle + "&body=" + encoded;

    // Toggle.
    toggle.addEventListener("click", function (e) {
      e.stopPropagation();

      if (activePopover && activePopover !== popover) {
        closeActive();
      }

      if (popover.hidden) {
        popover.hidden = false;
        toggle.setAttribute("aria-expanded", "true");
        activePopover = popover;
        activeToggle = toggle;
      } else {
        closeActive();
      }
    });

    // Copy link button.
    var copyBtn = popover.querySelector(".share-copy");
    if (copyBtn) {
      var copyIcon = copyBtn.querySelector(".share-copy-icon");
      var checkIcon = copyBtn.querySelector(".share-check-icon");
      var copyLabel = copyBtn.querySelector(".share-copy-label");
      var copiedLabel = copyBtn.querySelector(".share-copied-label");

      copyBtn.addEventListener("click", function () {
        copyToClipboard(permalink, function () {
          if (copyIcon) copyIcon.style.display = "none";
          if (checkIcon) checkIcon.style.display = "";
          if (copyLabel) copyLabel.hidden = true;
          if (copiedLabel) copiedLabel.hidden = false;
          copyBtn.classList.add("copied");

          setTimeout(function () {
            if (copyIcon) copyIcon.style.display = "";
            if (checkIcon) checkIcon.style.display = "none";
            if (copyLabel) copyLabel.hidden = false;
            if (copiedLabel) copiedLabel.hidden = true;
            copyBtn.classList.remove("copied");
          }, 2000);
        });
      });
    }
  });

  // Close on outside click.
  document.addEventListener("click", function (e) {
    if (activePopover) {
      var wrap = activePopover.closest(".share-wrap");
      if (!wrap || !wrap.contains(e.target)) {
        closeActive();
      }
    }
  });

  // Close on Escape.
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && activePopover) {
      var t = activeToggle;
      closeActive();
      if (t) t.focus();
    }
  });
})();
