/**
 * OSG Comments — OAuth2-authenticated threaded comments.
 *
 * Loaded only when config.comments_enabled is true.
 * Uses cookie-based sessions (osg_session) set by the auth flow.
 * All API calls include credentials for same-origin cookie auth.
 */
(function () {
  "use strict";

  var section = document.getElementById("osg-comments");
  if (!section) return;

  var apiURL = section.dataset.apiUrl || "";
  var pagePath = section.dataset.pagePath || window.location.pathname;

  var loginArea = section.querySelector(".comments-login");
  var form = section.querySelector(".comments-form");
  var formUserEl = section.querySelector(".comments-form-user");
  var textarea = section.querySelector(".comments-input");
  var submitBtn = section.querySelector(".comments-submit");
  var listEl = section.querySelector(".comments-list");
  var emptyMsg = section.querySelector(".comments-empty");

  var currentUser = null;

  // --- API helpers ---

  function apiGet(endpoint) {
    return fetch(apiURL + endpoint, {
      method: "GET",
      credentials: "include",
    }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    });
  }

  function apiPost(endpoint, body) {
    return fetch(apiURL + endpoint, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    });
  }

  function apiDelete(endpoint) {
    return fetch(apiURL + endpoint, {
      method: "DELETE",
      credentials: "include",
    }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      return r.json();
    });
  }

  // --- Rendering ---

  function escapeHTML(str) {
    var div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  function timeAgo(dateStr) {
    var date = new Date(dateStr);
    var now = new Date();
    var diff = Math.floor((now - date) / 1000);
    if (diff < 60) return "just now";
    if (diff < 3600) return Math.floor(diff / 60) + "m ago";
    if (diff < 86400) return Math.floor(diff / 3600) + "h ago";
    if (diff < 2592000) return Math.floor(diff / 86400) + "d ago";
    return date.toLocaleDateString();
  }

  function renderAvatar(author) {
    if (author && author.avatar_url) {
      return '<img class="comment-avatar" src="' + escapeHTML(author.avatar_url) + '" alt="" width="32" height="32" loading="lazy">';
    }
    return '<div class="comment-avatar comment-avatar-placeholder"></div>';
  }

  function renderComment(comment, depth) {
    if (depth === undefined) depth = 0;
    var maxIndent = 5;
    var indentClass = "comment-depth-" + Math.min(depth, maxIndent);
    var isDeleted = comment.body === "" && comment.author && comment.author.name === "";

    var html = '<div class="comment ' + indentClass + '" data-id="' + comment.id + '">';
    html += '<div class="comment-header">';
    html += renderAvatar(isDeleted ? null : comment.author);
    html += '<div class="comment-meta">';

    if (isDeleted) {
      html += '<span class="comment-author comment-deleted-author">' + escapeHTML(section.querySelector(".comments-empty").dataset.deletedLabel || "[comment deleted]") + '</span>';
    } else {
      html += '<span class="comment-author">' + escapeHTML(comment.author.name) + '</span>';
    }

    html += '<time class="comment-time" datetime="' + escapeHTML(comment.created_at) + '">' + timeAgo(comment.created_at) + '</time>';
    html += '</div></div>';

    if (isDeleted) {
      html += '<div class="comment-body comment-body-deleted"><em>' + escapeHTML(section.querySelector(".comments-empty").dataset.deletedLabel || "[comment deleted]") + '</em></div>';
    } else {
      html += '<div class="comment-body">' + escapeHTML(comment.body).replace(/\n/g, "<br>") + '</div>';
    }

    // Action buttons (only for non-deleted comments, authenticated users).
    if (!isDeleted && currentUser) {
      html += '<div class="comment-actions">';
      html += '<button class="comment-reply-btn" type="button" data-id="' + comment.id + '">Reply</button>';
      if (currentUser.id === comment.author_id || (comment.user_id && currentUser.id === comment.user_id)) {
        html += '<button class="comment-delete-btn" type="button" data-id="' + comment.id + '">Delete</button>';
      }
      html += '</div>';
    }

    // Reply form placeholder.
    html += '<div class="comment-reply-form" data-parent-id="' + comment.id + '" hidden></div>';

    // Render replies recursively.
    if (comment.replies && comment.replies.length > 0) {
      for (var i = 0; i < comment.replies.length; i++) {
        html += renderComment(comment.replies[i], depth + 1);
      }
    }

    html += '</div>';
    return html;
  }

  function renderComments(comments) {
    if (!comments || comments.length === 0) {
      if (emptyMsg) emptyMsg.hidden = false;
      listEl.innerHTML = "";
      listEl.appendChild(emptyMsg);
      return;
    }
    if (emptyMsg) emptyMsg.hidden = true;

    var html = "";
    for (var i = 0; i < comments.length; i++) {
      html += renderComment(comments[i], 0);
    }
    listEl.innerHTML = html;

    // Attach event listeners.
    attachCommentListeners();
  }

  function attachCommentListeners() {
    // Reply buttons.
    var replyBtns = listEl.querySelectorAll(".comment-reply-btn");
    for (var i = 0; i < replyBtns.length; i++) {
      replyBtns[i].addEventListener("click", handleReplyClick);
    }

    // Delete buttons.
    var deleteBtns = listEl.querySelectorAll(".comment-delete-btn");
    for (var j = 0; j < deleteBtns.length; j++) {
      deleteBtns[j].addEventListener("click", handleDeleteClick);
    }
  }

  function handleReplyClick(e) {
    var btn = e.currentTarget;
    var parentId = btn.dataset.id;
    var commentEl = btn.closest(".comment");
    var replyFormEl = commentEl.querySelector('.comment-reply-form[data-parent-id="' + parentId + '"]');

    if (!replyFormEl) return;

    // Toggle: if already showing, hide it.
    if (!replyFormEl.hidden) {
      replyFormEl.hidden = true;
      replyFormEl.innerHTML = "";
      return;
    }

    replyFormEl.hidden = false;
    replyFormEl.innerHTML =
      '<textarea class="comments-input comments-reply-input" rows="2" placeholder="Reply..." required></textarea>' +
      '<div class="comments-form-actions">' +
      '<button class="comments-submit comments-reply-submit" type="button">Reply</button>' +
      '<button class="comments-cancel" type="button">Cancel</button>' +
      '</div>';

    var replyTextarea = replyFormEl.querySelector(".comments-reply-input");
    var replySubmit = replyFormEl.querySelector(".comments-reply-submit");
    var replyCancel = replyFormEl.querySelector(".comments-cancel");

    replyTextarea.focus();

    replySubmit.addEventListener("click", function () {
      var body = replyTextarea.value.trim();
      if (!body) return;
      replySubmit.disabled = true;

      apiPost("/api/v1/comments", {
        page_path: pagePath,
        parent_id: parseInt(parentId, 10),
        body: body,
      })
        .then(function () {
          return loadComments();
        })
        .catch(function () {
          replySubmit.disabled = false;
        });
    });

    replyCancel.addEventListener("click", function () {
      replyFormEl.hidden = true;
      replyFormEl.innerHTML = "";
    });
  }

  function handleDeleteClick(e) {
    var btn = e.currentTarget;
    var commentId = btn.dataset.id;
    btn.disabled = true;

    apiDelete("/api/v1/comments/" + commentId)
      .then(function () {
        return loadComments();
      })
      .catch(function () {
        btn.disabled = false;
      });
  }

  // --- Auth UI ---

  function showLoginUI() {
    if (loginArea) loginArea.hidden = false;
    if (form) form.hidden = true;

    // Set return_to on login links.
    var loginBtns = section.querySelectorAll(".comments-login-btn");
    for (var i = 0; i < loginBtns.length; i++) {
      var provider = loginBtns[i].dataset.provider;
      loginBtns[i].href = apiURL + "/api/v1/auth/" + provider + "?return_to=" + encodeURIComponent(window.location.pathname + window.location.hash);
    }
  }

  function showAuthenticatedUI(user) {
    currentUser = user;
    if (loginArea) loginArea.hidden = true;
    if (form) form.hidden = false;

    if (formUserEl) {
      formUserEl.innerHTML =
        renderAvatar(user) +
        '<span class="comments-form-username">' + escapeHTML(user.name) + '</span>' +
        '<button class="comments-logout-btn" type="button">Logout</button>';

      var logoutBtn = formUserEl.querySelector(".comments-logout-btn");
      if (logoutBtn) {
        logoutBtn.addEventListener("click", function () {
          apiPost("/api/v1/auth/logout", {})
            .then(function () {
              currentUser = null;
              showLoginUI();
              loadComments();
            })
            .catch(function () {});
        });
      }
    }
  }

  // --- Load comments ---

  function loadComments() {
    return apiGet("/api/v1/comments?page=" + encodeURIComponent(pagePath))
      .then(function (data) {
        if (data.user) {
          showAuthenticatedUI(data.user);
        } else {
          showLoginUI();
        }
        renderComments(data.comments);
      })
      .catch(function () {
        // API not available — show login UI as fallback.
        showLoginUI();
      });
  }

  // --- Form submission ---

  if (form) {
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var body = textarea.value.trim();
      if (!body) return;
      submitBtn.disabled = true;

      apiPost("/api/v1/comments", {
        page_path: pagePath,
        body: body,
      })
        .then(function () {
          textarea.value = "";
          submitBtn.disabled = false;
          return loadComments();
        })
        .catch(function () {
          submitBtn.disabled = false;
        });
    });
  }

  // --- Init ---
  loadComments();
})();
