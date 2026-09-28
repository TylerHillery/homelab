(function () {
  "use strict";

  document.addEventListener("DOMContentLoaded", function () {
    if (!document.querySelector("[data-inventory-page]")) return;

    const host = document.querySelector("#inventory-drawer-host");
    let restoreFocus = null;

    function currentDrawer() {
      return host.querySelector(".inventory-drawer");
    }

    function showDrawer() {
      document.body.classList.add("inventory-panel-open");
      currentDrawer()?.querySelector("[data-inventory-close]")?.focus();
    }

    function hideDrawer(updateURL) {
      if (updateURL) {
        const url = new URL(window.location.href);
        url.searchParams.delete("product");
        url.searchParams.delete("asset");
        window.history.pushState(null, "", url);
      }
      host.replaceChildren();
      document.body.classList.remove("inventory-panel-open");
      if (restoreFocus?.isConnected) restoreFocus.focus();
      restoreFocus = null;
    }

    async function loadDrawer(url, updateURL) {
      const endpoint = new URL("/console/inventory/detail", window.location.origin);
      const type = url.searchParams.has("asset") ? "asset" : "product";
      endpoint.searchParams.set(type, url.searchParams.get(type));
      const response = await fetch(endpoint, {headers: {Accept: "text/html"}});
      if (!response.ok) throw new Error("Could not load inventory details");
      host.innerHTML = await response.text();
      if (updateURL) window.history.pushState(null, "", url);
      showDrawer();
    }

    document.addEventListener("click", function (event) {
      const link = event.target.closest("a[data-inventory-detail]");
      if (link) {
        event.preventDefault();
        const url = new URL(link.href);
        const current = new URL(window.location.href);
        if (current.searchParams.has("filter") && !url.searchParams.has("filter")) {
          url.searchParams.set("filter", current.searchParams.get("filter"));
        }
        restoreFocus = link;
        loadDrawer(url, true).catch(function () { window.location.assign(url.href); });
        return;
      }
      if (event.target.closest("[data-inventory-close]") || event.target.matches("[data-inventory-backdrop]")) {
        hideDrawer(true);
      }
    });

    document.addEventListener("keydown", function (event) {
      const drawer = currentDrawer();
      if (!drawer) return;
      if (event.key === "Escape") {
        event.preventDefault();
        hideDrawer(true);
        return;
      }
      if (event.key !== "Tab") return;
      const focusable = Array.from(drawer.querySelectorAll('a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])'));
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    });

    window.addEventListener("popstate", function () {
      const url = new URL(window.location.href);
      if (!url.searchParams.has("asset") && !url.searchParams.has("product")) {
        hideDrawer(false);
      } else {
        loadDrawer(url, false).catch(function () { window.location.reload(); });
      }
    });

    if (currentDrawer()) showDrawer();
  });
})();
