/* toc-spy.js — Scroll-spy for the sticky TOC sidebar.
   Highlights the current heading in the desktop TOC using IntersectionObserver. */
(function () {
  "use strict";

  var toc = document.querySelector(".toc-desktop");
  if (!toc) return;

  var links = toc.querySelectorAll(".toc-item a");
  if (!links.length) return;

  // Build a map of heading id -> toc item element.
  var itemMap = {};
  links.forEach(function (link) {
    var href = link.getAttribute("href");
    if (href && href.startsWith("#")) {
      itemMap[href.slice(1)] = link.closest(".toc-item");
    }
  });

  var headingIds = Object.keys(itemMap);
  var headings = [];
  headingIds.forEach(function (id) {
    var el = document.getElementById(id);
    if (el) headings.push(el);
  });

  if (!headings.length) return;

  var activeItem = null;

  function setActive(id) {
    if (activeItem) activeItem.classList.remove("toc-active");
    var item = itemMap[id];
    if (item) {
      item.classList.add("toc-active");
      activeItem = item;
    }
  }

  // Use IntersectionObserver with a rootMargin that triggers when a heading
  // crosses the top ~20% of the viewport.
  var observer = new IntersectionObserver(
    function (entries) {
      // Find the topmost visible heading.
      var visible = [];
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          visible.push(entry);
        }
      });

      if (visible.length > 0) {
        // Pick the one closest to the top.
        visible.sort(function (a, b) {
          return a.boundingClientRect.top - b.boundingClientRect.top;
        });
        setActive(visible[0].target.id);
      }
    },
    {
      rootMargin: "-10% 0px -80% 0px",
      threshold: 0,
    }
  );

  headings.forEach(function (h) {
    observer.observe(h);
  });
})();
