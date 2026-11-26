(function () {
  var root = document.documentElement;

  function labelThemes() {
    var dark = root.dataset.theme === "dark";
    document.querySelectorAll("[data-theme-toggle]").forEach(function (button) {
      var label = dark ? button.dataset.labelLight : button.dataset.labelDark;
      if (label) button.setAttribute("aria-label", label);
      button.setAttribute("aria-pressed", String(dark));
    });
  }

  labelThemes();
  document.querySelectorAll("[data-theme-toggle]").forEach(function (button) {
    button.addEventListener("click", function () {
      var next = root.dataset.theme === "dark" ? "light" : "dark";
      root.dataset.theme = next;
      try {
        localStorage.setItem("theme", next);
      } catch (error) {
        /* private mode */
      }
      labelThemes();
    });
  });

  function announce(text) {
    var live = document.getElementById("copied-live");
    if (live) live.textContent = text;
  }

  document.querySelectorAll("[data-copy]").forEach(function (button) {
    var text = button.getAttribute("data-copy");
    var original = button.getAttribute("aria-label");
    var copied = button.getAttribute("data-copied-label") || "Copied";

    function done() {
      button.classList.add("copied");
      button.setAttribute("aria-label", copied);
      announce(copied);
      setTimeout(function () {
        button.classList.remove("copied");
        if (original) button.setAttribute("aria-label", original);
      }, 2000);
    }

    function fallback() {
      var area = document.createElement("textarea");
      area.value = text;
      area.setAttribute("readonly", "");
      area.style.position = "fixed";
      area.style.top = "-1000px";
      area.style.opacity = "0";
      document.body.appendChild(area);
      area.select();
      try {
        document.execCommand("copy");
        done();
      } catch (error) {
        /* nothing else to try */
      }
      area.remove();
    }

    button.addEventListener("click", function () {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done, fallback);
      } else {
        fallback();
      }
    });
  });
})();
