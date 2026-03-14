/* sidenotes.js — Converts Goldmark footnotes into Tufte-style sidenotes
   on wide screens. Falls back to standard footnotes on narrow screens
   or when the TOC sidebar occupies the right margin. */
(function () {
  'use strict';

  var MIN_WIDTH = 1200;

  function init() {
    var section = document.querySelector('.footnotes');
    if (!section) return;

    // Don't convert to sidenotes when TOC is present (right margin occupied).
    var content = document.querySelector('.article-content');
    if (content && content.classList.contains('article-content--with-toc')) return;

    var refs = document.querySelectorAll('sup[id^="fnref:"] > a.footnote-ref');
    if (!refs.length) return;

    var created = 0;
    for (var i = 0; i < refs.length; i++) {
      var ref = refs[i];
      var noteId = ref.getAttribute('href');
      if (!noteId) continue;
      noteId = noteId.replace('#', '');

      var note = document.getElementById(noteId);
      if (!note) continue;

      var sidenote = document.createElement('span');
      sidenote.className = 'sidenote';
      sidenote.setAttribute('role', 'note');
      sidenote.setAttribute('aria-label', 'Sidenote ' + (i + 1));

      // Clone footnote content, strip backref link.
      var clone = note.cloneNode(true);
      var backref = clone.querySelector('.footnote-backref');
      if (backref) backref.remove();
      // Unwrap <p> inside <li> for cleaner display.
      var p = clone.querySelector('p');
      sidenote.innerHTML = p ? p.innerHTML : clone.innerHTML;

      var sup = ref.closest('sup');
      if (sup && sup.parentNode) {
        sup.parentNode.insertBefore(sidenote, sup.nextSibling);
        created++;
      }
    }

    if (created > 0 && content) {
      content.classList.add('has-sidenotes');
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
