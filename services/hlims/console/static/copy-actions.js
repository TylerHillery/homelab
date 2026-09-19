(function () {
  "use strict";

  function fallbackCopy(value) {
    const input = document.createElement("textarea");
    input.value = value;
    input.setAttribute("readonly", "");
    input.className = "clipboard-proxy";
    document.body.appendChild(input);
    let copied;
    try {
      input.select();
      copied = document.execCommand("copy");
    } finally {
      input.remove();
    }
    if (!copied) throw new Error("copy command was rejected");
  }

  async function copy(value) {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
      return;
    }
    fallbackCopy(value);
  }

  function updateSSHCommand(control) {
    const command = "ssh " + control.dataset.sshUser + "@" + control.dataset.sshHost;
    const copyButton = control.querySelector("[data-ssh-copy]");
    copyButton.dataset.copyValue = command;
    copyButton.setAttribute("aria-label", "Copy " + command);
  }

  document.addEventListener("click", function (event) {
    const userOption = event.target.closest("[data-ssh-user-option]");
    const hostOption = event.target.closest("[data-ssh-host-option]");
    const option = userOption || hostOption;
    if (!option) return;
    const control = option.closest("[data-ssh-control]");
    const kind = userOption ? "user" : "host";
    const value = userOption ? userOption.dataset.sshUserOption : hostOption.dataset.sshHostOption;
    control.dataset[kind === "user" ? "sshUser" : "sshHost"] = value;
    control.querySelector("[data-ssh-" + kind + "-label]").textContent = value;
    control.querySelectorAll("[data-ssh-" + kind + "-option]").forEach(function (button) {
      button.setAttribute("aria-pressed", String(button === option));
    });
    option.closest("details").removeAttribute("open");
    updateSSHCommand(control);
  });

  document.addEventListener("click", async function (event) {
    const button = event.target.closest("[data-copy-value]");
    if (!button) return;
    const label = button.querySelector("[data-copy-label]");
    const original = label.textContent;
    try {
      await copy(button.dataset.copyValue);
      label.textContent = "Copied";
      button.classList.add("is-copied");
    } catch (_) {
      label.textContent = "Failed";
      button.classList.add("is-copy-error");
    }
    window.setTimeout(function () {
      label.textContent = original;
      button.classList.remove("is-copied", "is-copy-error");
    }, 1800);
  });
})();
