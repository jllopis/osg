/**
 * OSG Lightbox — zero-dependency image lightbox with gallery navigation.
 *
 * Features:
 * - Click any figure[data-lightbox] image to open fullscreen overlay
 * - Keyboard navigation: Esc (close), Left/Right (prev/next)
 * - Touch swipe support on mobile
 * - Counter showing current/total when multiple images on page
 * - Respects prefers-reduced-motion
 * - Nord-styled overlay consistent with theme
 */
(function () {
  "use strict";

  var overlay, imgEl, captionEl, counterEl, prevBtn, nextBtn;
  var images = [];
  var currentIndex = 0;
  var touchStartX = 0;

  function init() {
    // Collect all lightbox-enabled figures.
    var figures = document.querySelectorAll("figure[data-lightbox]");
    if (!figures.length) return;

    images = [];
    figures.forEach(function (fig) {
      var img = fig.querySelector("img");
      if (!img) return;
      var caption = fig.querySelector("figcaption");
      images.push({
        src: img.src,
        alt: img.alt || "",
        caption: caption ? caption.textContent : "",
      });
    });

    if (!images.length) return;

    // Group consecutive figures into gallery wrappers.
    groupGalleries(figures);

    // Build overlay DOM.
    buildOverlay();

    // Attach click handlers to all figures.
    figures.forEach(function (fig, i) {
      fig.addEventListener("click", function (e) {
        e.preventDefault();
        open(i);
      });
    });
  }

  function groupGalleries(figures) {
    var consecutive = [];
    var groups = [];

    figures.forEach(function (fig, i) {
      if (
        consecutive.length === 0 ||
        isConsecutiveFigure(consecutive[consecutive.length - 1], fig)
      ) {
        consecutive.push(fig);
      } else {
        if (consecutive.length >= 2) groups.push(consecutive.slice());
        consecutive = [fig];
      }
    });
    if (consecutive.length >= 2) groups.push(consecutive);

    groups.forEach(function (group) {
      var parent = group[0].parentNode;
      var gallery = document.createElement("div");
      gallery.className = "gallery";
      parent.insertBefore(gallery, group[0]);
      group.forEach(function (fig) {
        gallery.appendChild(fig);
      });
    });
  }

  function isConsecutiveFigure(a, b) {
    // Two figures are consecutive if b comes right after a,
    // ignoring whitespace-only text nodes between them.
    var node = a.nextSibling;
    while (node) {
      if (node === b) return true;
      if (node.nodeType === 3 && node.textContent.trim() === "") {
        node = node.nextSibling;
        continue;
      }
      return false;
    }
    return false;
  }

  function buildOverlay() {
    overlay = document.createElement("div");
    overlay.className = "lightbox-overlay";
    overlay.setAttribute("role", "dialog");
    overlay.setAttribute("aria-modal", "true");
    overlay.setAttribute("aria-label", "Image viewer");
    overlay.setAttribute("data-count", images.length);

    imgEl = document.createElement("img");
    imgEl.className = "lightbox-img";

    captionEl = document.createElement("div");
    captionEl.className = "lightbox-caption";

    counterEl = document.createElement("div");
    counterEl.className = "lightbox-counter";

    var closeBtn = document.createElement("button");
    closeBtn.className = "lightbox-close";
    closeBtn.setAttribute("aria-label", "Close");
    closeBtn.innerHTML = "&#215;";
    closeBtn.addEventListener("click", function (e) {
      e.stopPropagation();
      close();
    });

    prevBtn = document.createElement("button");
    prevBtn.className = "lightbox-nav lightbox-prev";
    prevBtn.setAttribute("aria-label", "Previous image");
    prevBtn.innerHTML = "&#8249;";
    prevBtn.addEventListener("click", function (e) {
      e.stopPropagation();
      navigate(-1);
    });

    nextBtn = document.createElement("button");
    nextBtn.className = "lightbox-nav lightbox-next";
    nextBtn.setAttribute("aria-label", "Next image");
    nextBtn.innerHTML = "&#8250;";
    nextBtn.addEventListener("click", function (e) {
      e.stopPropagation();
      navigate(1);
    });

    overlay.appendChild(imgEl);
    overlay.appendChild(captionEl);
    overlay.appendChild(counterEl);
    overlay.appendChild(closeBtn);
    overlay.appendChild(prevBtn);
    overlay.appendChild(nextBtn);

    // Close on overlay background click.
    overlay.addEventListener("click", function (e) {
      if (e.target === overlay) close();
    });

    // Touch swipe support.
    overlay.addEventListener(
      "touchstart",
      function (e) {
        touchStartX = e.changedTouches[0].screenX;
      },
      { passive: true }
    );

    overlay.addEventListener(
      "touchend",
      function (e) {
        var dx = e.changedTouches[0].screenX - touchStartX;
        if (Math.abs(dx) > 50) {
          navigate(dx < 0 ? 1 : -1);
        }
      },
      { passive: true }
    );

    document.body.appendChild(overlay);

    // Keyboard handler.
    document.addEventListener("keydown", function (e) {
      if (!overlay.classList.contains("active")) return;
      if (e.key === "Escape") close();
      else if (e.key === "ArrowLeft") navigate(-1);
      else if (e.key === "ArrowRight") navigate(1);
    });
  }

  function show(index) {
    var img = images[index];
    imgEl.src = img.src;
    imgEl.alt = img.alt;
    captionEl.textContent = img.caption;
    captionEl.style.display = img.caption ? "" : "none";
    counterEl.textContent = index + 1 + " / " + images.length;
    currentIndex = index;
  }

  function open(index) {
    show(index);
    overlay.classList.add("active");
    document.body.style.overflow = "hidden";
  }

  function close() {
    overlay.classList.remove("active");
    document.body.style.overflow = "";
  }

  function navigate(dir) {
    if (images.length <= 1) return;
    var next = (currentIndex + dir + images.length) % images.length;
    show(next);
  }

  // Initialize when DOM is ready.
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
