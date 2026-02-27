/* OSG Reading Progress Bar
   Thin bar at the top of the page showing scroll progress through article content. */
(function () {
  "use strict";
  var bar = document.querySelector(".reading-progress-bar");
  if (!bar) return;
  var article = document.querySelector(".article-content");
  if (!article) return;

  function update() {
    var rect = article.getBoundingClientRect();
    var top = -rect.top;
    var height = rect.height - window.innerHeight;
    var pct = height > 0 ? Math.min(Math.max(top / height, 0), 1) * 100 : 0;
    bar.style.width = pct + "%";
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
