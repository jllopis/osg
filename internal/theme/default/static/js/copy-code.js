/* copy-code.js — Adds a "Copy" button to every <pre><code> block.
   Zero dependencies, vanilla JS. */
(function () {
  'use strict';

  var ICON_COPY =
    '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' +
    '<rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>' +
    '<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';

  var ICON_CHECK =
    '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' +
    '<polyline points="20 6 9 17 4 12"/></svg>';

  function init() {
    var blocks = document.querySelectorAll('pre > code');
    for (var i = 0; i < blocks.length; i++) {
      attach(blocks[i]);
    }
  }

  function attach(code) {
    var pre = code.parentElement;
    if (!pre || pre.querySelector('.code-copy-btn')) return;

    var btn = document.createElement('button');
    btn.className = 'code-copy-btn';
    btn.type = 'button';
    btn.setAttribute('aria-label', 'Copy code');
    btn.innerHTML = ICON_COPY;

    btn.addEventListener('click', function () {
      var text = code.textContent;
      navigator.clipboard.writeText(text).then(function () {
        btn.classList.add('copied');
        btn.innerHTML = ICON_CHECK;
        setTimeout(function () {
          btn.classList.remove('copied');
          btn.innerHTML = ICON_COPY;
        }, 2000);
      });
    });

    pre.appendChild(btn);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
