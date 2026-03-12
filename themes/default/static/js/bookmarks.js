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
      image: data.image || "",
      reading_time: data.reading_time || "",
      taxonomies: data.taxonomies || {},
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

  /* --- Collect taxonomy data-tax-* attributes from a button. --- */
  function collectTaxonomies(btn) {
    var taxonomies = {};
    var attrs = btn.attributes;
    for (var i = 0; i < attrs.length; i++) {
      var name = attrs[i].name;
      if (name.indexOf("data-tax-") === 0) {
        var kind = name.substring(9);
        var val = attrs[i].value;
        if (val) taxonomies[kind] = val.split(",");
      }
    }
    return taxonomies;
  }

  /* --- Refresh visual state of all bookmark buttons for a given path. --- */
  function refreshButtons(path) {
    var btns = document.querySelectorAll('.bookmark-btn[data-path="' + CSS.escape(path) + '"]');
    var saved = isBookmarked(path);
    btns.forEach(function (btn) {
      btn.setAttribute("aria-pressed", saved ? "true" : "false");
      btn.classList.toggle("bookmark-btn--saved", saved);
      btn.title = saved
        ? (btn.dataset.labelRemove || "Remove bookmark")
        : (btn.dataset.labelSave || "Save for later");
      var label = btn.querySelector(".bookmark-label");
      if (label) {
        label.textContent = btn.title;
      }
    });
  }

  /* --- Initialize all bookmark buttons on the page. --- */
  function initBookmarkButtons() {
    var btns = document.querySelectorAll(".bookmark-btn");
    if (!btns.length) return;

    // Track paths we've already set up listeners for refresh.
    var initialised = {};

    btns.forEach(function (btn) {
      var path = btn.dataset.path;
      if (!path) return;

      // Initial visual state.
      if (!initialised[path]) {
        refreshButtons(path);
        initialised[path] = true;
      } else {
        // Just sync this button.
        var saved = isBookmarked(path);
        btn.setAttribute("aria-pressed", saved ? "true" : "false");
        btn.classList.toggle("bookmark-btn--saved", saved);
        btn.title = saved
          ? (btn.dataset.labelRemove || "Remove bookmark")
          : (btn.dataset.labelSave || "Save for later");
      }

      btn.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        if (isBookmarked(path)) {
          removeBookmark(path);
        } else {
          addBookmark({
            path: path,
            title: btn.dataset.title || "",
            summary: btn.dataset.summary || "",
            date: btn.dataset.date || "",
            image: btn.dataset.image || "",
            reading_time: btn.dataset.readingTime || "",
            taxonomies: collectTaxonomies(btn)
          });
        }
        refreshButtons(path);
      });
    });
  }

  /* --- Bookmarks page: render saved list as cards --- */
  function initBookmarksPage() {
    var container = document.querySelector(".bookmarks-list");
    if (!container) return;

    var emptyText = container.dataset.emptyText || "No bookmarks yet.";
    var removeText = container.dataset.removeText || "Remove";
    var minReadText = container.dataset.minReadText || "min read";

    function render() {
      var bookmarks = load();

      if (bookmarks.length === 0) {
        container.innerHTML = '<p class="bookmarks-empty">' + escapeHTML(emptyText) + "</p>";
        return;
      }

      container.innerHTML = "";
      bookmarks.forEach(function (b) {
        var article = document.createElement("article");
        article.className = "card";

        // Image
        if (b.image) {
          var imgWrap = document.createElement("div");
          imgWrap.className = "card-image";
          var img = document.createElement("img");
          img.src = b.image;
          img.alt = b.title;
          img.loading = "lazy";
          imgWrap.appendChild(img);
          article.appendChild(imgWrap);
        }

        var content = document.createElement("div");
        content.className = "card-content";

        // Title
        var h3 = document.createElement("h3");
        var link = document.createElement("a");
        link.href = b.path;
        link.textContent = b.title;
        h3.appendChild(link);
        content.appendChild(h3);

        // Meta (date + reading time)
        var meta = document.createElement("div");
        meta.className = "meta";
        if (b.date) {
          var metaLine = document.createElement("span");
          metaLine.className = "meta-line";
          var time = document.createElement("time");
          time.setAttribute("datetime", b.date);
          time.textContent = b.date;
          metaLine.appendChild(time);
          meta.appendChild(metaLine);
        }
        if (b.reading_time) {
          var badge = document.createElement("span");
          badge.className = "reading-badge";
          badge.textContent = b.reading_time + " " + minReadText;
          meta.appendChild(badge);
        }
        if (b.date || b.reading_time) {
          content.appendChild(meta);
        }

        // Summary
        if (b.summary) {
          var p = document.createElement("p");
          p.className = "summary";
          p.textContent = b.summary;
          content.appendChild(p);
        }

        // Footer: pills + remove button
        var footer = document.createElement("div");
        footer.className = "card-footer";

        // Taxonomy pills
        var taxonomies = b.taxonomies || {};
        var hasTax = false;
        for (var k in taxonomies) {
          if (taxonomies.hasOwnProperty(k) && taxonomies[k].length) { hasTax = true; break; }
        }
        if (hasTax) {
          var pills = document.createElement("div");
          pills.className = "pills";
          for (var kind in taxonomies) {
            if (!taxonomies.hasOwnProperty(kind)) continue;
            taxonomies[kind].forEach(function (term) {
              var pill = document.createElement("span");
              pill.className = "pill pill--default";
              pill.textContent = term;
              pills.appendChild(pill);
            });
          }
          footer.appendChild(pills);
        }

        var removeBtn = document.createElement("button");
        removeBtn.className = "bookmark-remove-btn";
        removeBtn.type = "button";
        removeBtn.title = removeText;
        removeBtn.setAttribute("aria-label", removeText);
        removeBtn.innerHTML = '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M19 6l-.7 11.3a2 2 0 0 1-2 1.7H7.7a2 2 0 0 1-2-1.7L5 6"/></svg>';
        removeBtn.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          removeBookmark(b.path);
          render();
        });

        footer.appendChild(removeBtn);
        content.appendChild(footer);
        article.appendChild(content);
        container.appendChild(article);
      });
    }

    render();
  }

  function escapeHTML(str) {
    var d = document.createElement("div");
    d.textContent = str;
    return d.innerHTML;
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
  initBookmarkButtons();
  initBookmarksPage();
  initExportImport();
})();
