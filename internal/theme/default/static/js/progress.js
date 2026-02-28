/* OSG Reading Progress Bar
   Thin bar at the bottom edge of the sticky header showing scroll progress. */
(function () {
  "use strict";
  var bar = document.querySelector(".reading-progress-bar");
  if (!bar) return;
  var article = document.querySelector(".article-content");
  if (!article) return;

  function getHeaderBottom() {
    var header = document.querySelector(".site-header");
    return header ? header.getBoundingClientRect().bottom : 0;
  }

  function update() {
    var rect = article.getBoundingClientRect();
    var top = -rect.top;
    var height = rect.height - window.innerHeight;
    var pct = height > 0 ? Math.min(Math.max(top / height, 0), 1) * 100 : 0;
    bar.style.width = pct + "%";
    bar.style.top = getHeaderBottom() + "px";
  }

  var ticking = false;
  window.addEventListener("scroll", function () {
    if (!ticking) {
      requestAnimationFrame(function () {
        update();
        ticking = false;
      });
      ticking = true;
    }
  });
  update();
})();
