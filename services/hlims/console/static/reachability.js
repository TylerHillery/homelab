(function () {
  "use strict";

  const probeTimeout = 5000;
  const refreshInterval = 30000;
  const active = new Map();

  function selector(kind, id) {
    return `[data-reachability-kind="${CSS.escape(kind)}"][data-reachability-id="${CSS.escape(id)}"]`;
  }

  function update(kind, id, status, detail) {
    const label = status.charAt(0).toUpperCase() + status.slice(1);
    document.querySelectorAll(selector(kind, id)).forEach(function (badge) {
      badge.className = `reachability-badge is-${status}`;
      badge.setAttribute("aria-label", `${badge.querySelector("strong").textContent}: ${label.toLowerCase()}. ${detail}`);
      badge.querySelector("[data-reachability-status]").textContent = label.toUpperCase();
      badge.querySelector("[data-reachability-detail]").textContent = detail;
      badge.querySelector("[data-reachability-a11y]").textContent = label;
    });
  }

  async function probe(kind, id, url) {
    const key = `${kind}:${id}`;
    if (active.has(key)) return active.get(key);
    update(kind, id, "checking", "Testing this route from the current browser");

    const promise = (async function () {
      const controller = new AbortController();
      const timer = window.setTimeout(function () { controller.abort(); }, probeTimeout);
      const started = performance.now();
      try {
        await fetch(url, {cache: "no-store", mode: "no-cors", redirect: "follow", signal: controller.signal});
        const duration = Math.max(1, Math.round(performance.now() - started));
        update(kind, id, "reachable", `Reached from this browser in ${duration} ms at ${new Date().toLocaleTimeString()}`);
      } catch (error) {
        const reason = error.name === "AbortError" ? "The browser probe timed out" : "The browser could not reach this route";
        update(kind, id, "unreachable", `${reason}. Network, TLS, mixed-content, or browser policy may be the cause.`);
      } finally {
        window.clearTimeout(timer);
        active.delete(key);
      }
    })();
    active.set(key, promise);
    return promise;
  }

  function refresh() {
    const targets = new Map();
    document.querySelectorAll("[data-reachability-url]").forEach(function (badge) {
      targets.set(`${badge.dataset.reachabilityKind}:${badge.dataset.reachabilityId}`, badge);
    });
    targets.forEach(function (badge) {
      probe(badge.dataset.reachabilityKind, badge.dataset.reachabilityId, badge.dataset.reachabilityUrl);
    });
  }

  window.addEventListener("DOMContentLoaded", refresh);
  document.addEventListener("htmx:after:process", refresh);
  window.setInterval(refresh, refreshInterval);
})();
