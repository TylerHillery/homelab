(() => {
  const root = document.documentElement;
  const button = document.querySelector("[data-theme-toggle]");
  const storedTheme = localStorage.getItem("hlims-theme");

  if (storedTheme === "light" || storedTheme === "dark") {
    root.dataset.theme = storedTheme;
  }

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
    localStorage.setItem("hlims-theme", nextTheme);
    updateLabel();
  });
})();
