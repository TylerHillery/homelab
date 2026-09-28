(function () {
  "use strict";

  document.addEventListener("toggle", function (event) {
    const picker = event.target;
    if (!(picker instanceof HTMLDetailsElement) || !picker.matches(".route-picker, .machine-addresses")) return;

    picker.classList.remove("opens-up");
    if (!picker.open) return;

    requestAnimationFrame(function () {
      if (!picker.isConnected || !picker.open) return;
      const menu = picker.querySelector(".route-menu, .access-list");
      const anchor = picker.querySelector("summary").getBoundingClientRect();
      const below = Math.max(0, window.innerHeight - anchor.bottom - 12);
      const above = Math.max(0, anchor.top - 12);

      menu.style.maxHeight = "";
      if (menu.scrollHeight > below && above > below) {
        picker.classList.add("opens-up");
      }
      menu.style.maxHeight = `${picker.classList.contains("opens-up") ? above : below}px`;
    });
  }, true);
})();
