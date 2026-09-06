/* Buchfink — Projektseite
 *
 * Das einzige Skript der Seite. Es tut zwei Dinge: es schließt ein offenes
 * Menü in der Kopfleiste, und es öffnet die Dialoge, in denen die
 * Vertiefungen stecken. Die Navigation selbst funktioniert ohne das Skript,
 * die Dialoge nicht — für sie steht in jeder Seite ein noscript-Block, der
 * ihren Inhalt stattdessen in den Textfluss stellt. */
(function () {
  var menus = document.querySelectorAll('.topbar details');

  function closeMenus(except) {
    menus.forEach(function (menu) {
      if (menu !== except) menu.open = false;
    });
  }

  menus.forEach(function (menu) {
    menu.addEventListener('toggle', function () {
      if (menu.open) closeMenus(menu);
    });
  });

  document.addEventListener('click', function (event) {
    if (!event.target.closest('.topbar details')) closeMenus(null);
  });

  document.addEventListener('keydown', function (event) {
    if (event.key !== 'Escape') return;
    menus.forEach(function (menu) {
      if (!menu.open) return;
      menu.open = false;
      var summary = menu.querySelector('summary');
      if (summary) summary.focus();
    });
  });

  /* Dialoge */

  document.querySelectorAll('[data-dialog]').forEach(function (trigger) {
    var dialog = document.getElementById(trigger.getAttribute('data-dialog'));
    if (!dialog) return;
    trigger.addEventListener('click', function () {
      dialog.showModal();
    });
  });

  document.querySelectorAll('dialog.modal').forEach(function (dialog) {
    var close = dialog.querySelector('[data-close]');
    if (close) {
      close.addEventListener('click', function () {
        dialog.close();
      });
    }
    /* Ein Klick neben den Kasten schließt. Der Dialog hat keine Polsterung,
       also trifft ein Klick auf ihn selbst nur den Hintergrund; sicherer ist
       der Vergleich mit seinen Maßen. */
    dialog.addEventListener('click', function (event) {
      var box = dialog.getBoundingClientRect();
      var inside =
        box.top <= event.clientY &&
        event.clientY <= box.bottom &&
        box.left <= event.clientX &&
        event.clientX <= box.right;
      if (!inside) dialog.close();
    });
  });
})();
