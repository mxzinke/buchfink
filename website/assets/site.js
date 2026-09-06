/* Buchfink — Projektseite
 *
 * Das einzige Skript der Seite. Die Navigation funktioniert ohne es: die
 * Menüs sind <details>-Elemente und klappen von selbst auf. Was hier steht,
 * schließt ein offenes Menü wieder — beim Klick daneben, mit Escape und
 * sobald ein zweites aufgeht. */
(function () {
  var menus = document.querySelectorAll('.topbar details');
  if (!menus.length) return;

  function closeAll(except) {
    menus.forEach(function (menu) {
      if (menu !== except) menu.open = false;
    });
  }

  menus.forEach(function (menu) {
    menu.addEventListener('toggle', function () {
      if (menu.open) closeAll(menu);
    });
  });

  document.addEventListener('click', function (event) {
    if (!event.target.closest('.topbar details')) closeAll(null);
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
})();
