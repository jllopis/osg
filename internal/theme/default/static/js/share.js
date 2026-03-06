/**
 * OSG Share — Compact share button with popover dropdown.
 *
 * A single "Share" button toggles a popover with social sharing options
 * (X, LinkedIn, Bluesky, Email, Copy link). Closes on outside click or
 * Escape key.
 */
(function () {
  "use strict";

  // --- Helpers ---

  /** Resolve a possibly-relative path to an absolute URL. */
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

  // --- Share popover ---

  var wrap = document.querySelector(".share-wrap");
  if (!wrap) return;

  var toggle = wrap.querySelector(".share-toggle");
  var popover = wrap.querySelector(".share-popover");
  if (!toggle || !popover) return;

  var permalink = resolveURL(wrap.dataset.permalink || window.location.pathname);
  var title = wrap.dataset.title || document.title;
  var encoded = encodeURIComponent(permalink);
  var encodedTitle = encodeURIComponent(title);

  // --- Build share URLs ---

  var xBtn = popover.querySelector(".share-x");
  if (xBtn) {
    xBtn.href = "https://x.com/intent/tweet?url=" + encoded + "&text=" + encodedTitle;
  }

  var liBtn = popover.querySelector(".share-linkedin");
  if (liBtn) {
    liBtn.href = "https://www.linkedin.com/sharing/share-offsite/?url=" + encoded;
  }

  var bsBtn = popover.querySelector(".share-bluesky");
  if (bsBtn) {
    bsBtn.href = "https://bsky.app/intent/compose?text=" + encodedTitle + " " + encoded;
  }

  var emailBtn = popover.querySelector(".share-email");
  if (emailBtn) {
    emailBtn.href = "mailto:?subject=" + encodedTitle + "&body=" + encoded;
  }

  // --- Popover toggle ---

  function openPopover() {
    popover.hidden = false;
    toggle.setAttribute("aria-expanded", "true");
  }

  function closePopover() {
    popover.hidden = true;
    toggle.setAttribute("aria-expanded", "false");
  }

  toggle.addEventListener("click", function (e) {
    e.stopPropagation();
    if (popover.hidden) {
      openPopover();
    } else {
      closePopover();
    }
  });

  // Close on outside click.
  document.addEventListener("click", function (e) {
    if (!popover.hidden && !wrap.contains(e.target)) {
      closePopover();
    }
  });

  // Close on Escape.
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && !popover.hidden) {
      closePopover();
      toggle.focus();
    }
  });

  // --- Copy link button ---

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
})();
