(() => {
  const root = document.documentElement;
  try {
    const storedTheme = localStorage.getItem("hlims-theme");
    if (storedTheme === "light" || storedTheme === "dark") {
      root.dataset.theme = storedTheme;
    }
  } catch (_) {
    // The system preference remains available if browser storage is disabled.
  }

  function initializeToggle() {
    const button = document.querySelector("[data-theme-toggle]");
    if (!button) return;

    const activeTheme = () => root.dataset.theme || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
    const updateLabel = () => {
      const nextTheme = activeTheme() === "dark" ? "light" : "dark";
      button.setAttribute("aria-label", `Switch to ${nextTheme} theme`);
      button.setAttribute("title", `Switch to ${nextTheme} theme`);
    };

    updateLabel();
    button.addEventListener("click", () => {
      const nextTheme = activeTheme() === "dark" ? "light" : "dark";
      root.dataset.theme = nextTheme;
      try {
        localStorage.setItem("hlims-theme", nextTheme);
      } catch (_) {
        // The theme still changes for this page when storage is disabled.
      }
      updateLabel();
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initializeToggle, {once: true});
  } else {
    initializeToggle();
  }
})();
