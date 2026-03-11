/* bookmarks.js — "Save for later" reading list using localStorage.
   Allows users to bookmark posts and view them on /bookmarks/. */
(function () {
  "use strict";

  var STORAGE_KEY = "osg-bookmarks";

  function load() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY)) || [];
    } catch (_) {
      return [];
    }
  }

  function save(bookmarks) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(bookmarks));
    updateBadge();
  }

  function isBookmarked(path) {
    return load().some(function (b) { return b.path === path; });
  }

  function addBookmark(data) {
    var bookmarks = load();
    if (bookmarks.some(function (b) { return b.path === data.path; })) return;
    bookmarks.unshift({
      path: data.path,
      title: data.title,
      summary: data.summary || "",
      date: data.date || "",
      saved_at: new Date().toISOString()
    });
    save(bookmarks);
  }

  function removeBookmark(path) {
    var bookmarks = load().filter(function (b) { return b.path !== path; });
    save(bookmarks);
  }

  function updateBadge() {
    var badges = document.querySelectorAll(".bookmarks-badge");
    var count = load().length;
    badges.forEach(function (badge) {
      badge.textContent = count;
      badge.hidden = count === 0;
    });
  }

  /* --- Article page: save/remove button --- */
  function initArticleButton() {
    var btn = document.querySelector(".bookmark-btn");
    if (!btn) return;

    var path = btn.dataset.path;
    if (!path) return;

    function refresh() {
      var saved = isBookmarked(path);
      btn.setAttribute("aria-pressed", saved ? "true" : "false");
      btn.classList.toggle("bookmark-btn--saved", saved);
      var label = btn.querySelector(".bookmark-label");
      if (label) {
        label.textContent = saved
          ? (btn.dataset.labelRemove || "Remove bookmark")
          : (btn.dataset.labelSave || "Save for later");
      }
    }

    btn.addEventListener("click", function () {
      if (isBookmarked(path)) {
        removeBookmark(path);
      } else {
        addBookmark({
          path: path,
          title: btn.dataset.title || "",
          summary: btn.dataset.summary || "",
          date: btn.dataset.date || ""
        });
      }
      refresh();
    });

    refresh();
  }

  /* --- Bookmarks page: render saved list --- */
  function initBookmarksPage() {
    var container = document.querySelector(".bookmarks-list");
    if (!container) return;

    function render() {
      var bookmarks = load();
      var empty = container.querySelector(".bookmarks-empty");

      if (bookmarks.length === 0) {
        container.innerHTML = "";
        if (empty) container.appendChild(empty);
        else {
          var p = document.createElement("p");
          p.className = "bookmarks-empty";
          p.textContent = container.dataset.emptyText || "No bookmarks yet.";
          container.appendChild(p);
        }
        return;
      }

      container.innerHTML = "";
      bookmarks.forEach(function (b) {
        var item = document.createElement("a");
        item.className = "post-item";
        item.href = b.path;

        var body = document.createElement("div");
        body.className = "post-item-body";

        if (b.date) {
          var date = document.createElement("div");
          date.className = "post-item-date";
          var time = document.createElement("time");
          time.textContent = b.date;
          date.appendChild(time);
          body.appendChild(date);
        }

        var title = document.createElement("h3");
        title.className = "post-item-title";
        title.textContent = b.title;
        body.appendChild(title);

        if (b.summary) {
          var summary = document.createElement("p");
          summary.className = "post-item-summary";
          summary.textContent = b.summary;
          body.appendChild(summary);
        }

        var removeBtn = document.createElement("button");
        removeBtn.className = "bookmark-remove-btn";
        removeBtn.type = "button";
        removeBtn.textContent = "×";
        removeBtn.title = container.dataset.removeText || "Remove";
        removeBtn.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          removeBookmark(b.path);
          render();
        });

        item.appendChild(body);
        item.appendChild(removeBtn);
        container.appendChild(item);
      });
    }

    render();
  }

  /* --- Export / Import --- */
  function initExportImport() {
    var exportBtn = document.querySelector(".bookmarks-export");
    var importBtn = document.querySelector(".bookmarks-import");
    var importInput = document.querySelector(".bookmarks-import-file");

    if (exportBtn) {
      exportBtn.addEventListener("click", function () {
        var data = JSON.stringify(load(), null, 2);
        var blob = new Blob([data], { type: "application/json" });
        var url = URL.createObjectURL(blob);
        var a = document.createElement("a");
        a.href = url;
        a.download = "osg-bookmarks.json";
        a.click();
        URL.revokeObjectURL(url);
      });
    }

    if (importBtn && importInput) {
      importBtn.addEventListener("click", function () {
        importInput.click();
      });
      importInput.addEventListener("change", function () {
        var file = importInput.files[0];
        if (!file) return;
        var reader = new FileReader();
        reader.onload = function () {
          try {
            var imported = JSON.parse(reader.result);
            if (!Array.isArray(imported)) return;
            var existing = load();
            var paths = {};
            existing.forEach(function (b) { paths[b.path] = true; });
            imported.forEach(function (b) {
              if (b.path && !paths[b.path]) {
                existing.push(b);
                paths[b.path] = true;
              }
            });
            save(existing);
            initBookmarksPage();
          } catch (_) { /* ignore invalid files */ }
        };
        reader.readAsText(file);
      });
    }
  }

  updateBadge();
  initArticleButton();
  initBookmarksPage();
  initExportImport();
})();
