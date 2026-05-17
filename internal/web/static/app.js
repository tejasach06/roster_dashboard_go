(() => {
  const root = document.documentElement;
  const savedTheme = localStorage.getItem("theme");
  if (savedTheme) root.dataset.theme = savedTheme;
  document.querySelectorAll("[data-theme-toggle]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const next = root.dataset.theme === "dark" ? "light" : "dark";
      root.dataset.theme = next;
      localStorage.setItem("theme", next);
    });
  });

  document.querySelectorAll("[data-sidebar-toggle]").forEach((btn) => {
    btn.addEventListener("click", () => document.getElementById("sidebar")?.classList.toggle("open"));
  });

  document.querySelectorAll("[data-tab]").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".tab,.tab-panel").forEach((el) => el.classList.remove("active"));
      btn.classList.add("active");
      document.getElementById(btn.dataset.tab)?.classList.add("active");
    });
  });

  document.querySelectorAll("[data-modal-open]").forEach((btn) => {
    btn.addEventListener("click", () => document.getElementById(btn.dataset.modalOpen)?.showModal());
  });
  document.querySelectorAll("[data-modal-close]").forEach((btn) => {
    btn.addEventListener("click", () => btn.closest("dialog")?.close());
  });

  document.querySelectorAll("[data-fill-form]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const dialog = document.getElementById(btn.dataset.fillForm);
      if (!dialog) return;
      dialog.querySelectorAll("input,select").forEach((field) => {
        if (field.name && btn.dataset[field.name] !== undefined) field.value = btn.dataset[field.name] || "";
      });
      dialog.showModal();
    });
  });

  const roster = document.querySelector(".roster-page");
  const popover = document.getElementById("cell-popover");
  document.querySelector("[data-edit-toggle]")?.addEventListener("click", (event) => {
    roster?.classList.toggle("editing");
    event.currentTarget.textContent = roster?.classList.contains("editing") ? "Viewing mode" : "Edit mode";
  });
  document.querySelectorAll("[data-cell]").forEach((cell) => {
    cell.addEventListener("click", () => {
      if (!roster?.classList.contains("editing") || !popover) return;
      const form = popover.querySelector("form");
      form.elements.entry_id.value = cell.dataset.entryId || "";
      form.elements.employee_id.value = cell.dataset.employeeId || "";
      form.elements.date.value = cell.dataset.date || "";
      const rect = cell.getBoundingClientRect();
      popover.style.left = `${Math.min(rect.left, window.innerWidth - 260)}px`;
      popover.style.top = `${rect.bottom + 6}px`;
      popover.classList.remove("hidden");
    });
  });
  document.addEventListener("click", (event) => {
    if (!popover || popover.classList.contains("hidden")) return;
    if (!popover.contains(event.target) && !event.target.matches("[data-cell]")) popover.classList.add("hidden");
  });

  const search = document.querySelector("[data-roster-search]");
  const shift = document.querySelector("[data-shift-filter]");
  const filterRows = () => {
    const q = (search?.value || "").toLowerCase();
    const code = shift?.value || "";
    document.querySelectorAll("[data-roster-row]").forEach((row) => {
      const matchesText = !q || row.dataset.search.toLowerCase().includes(q);
      const matchesShift = !code || [...row.querySelectorAll("[data-cell]")].some((cell) => cell.dataset.shift === code);
      row.hidden = !(matchesText && matchesShift);
    });
  };
  search?.addEventListener("input", filterRows);
  shift?.addEventListener("change", filterRows);

  const parseCSV = (text) => {
    const lines = text.trim().split(/\r?\n/).filter(Boolean);
    if (lines.length < 2) return [];
    const headers = lines[0].split(",").map((h) => h.trim().replace(/^"|"$/g, "").toLowerCase());
    return lines.slice(1).map((line) => {
      const vals = line.split(",").map((v) => v.trim().replace(/^"|"$/g, ""));
      const row = {};
      headers.forEach((h, i) => row[h] = vals[i] || "");
      return row;
    });
  };
  document.querySelectorAll("[data-import-run]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const card = btn.closest("[data-import]");
      const output = card.querySelector("pre");
      const rows = parseCSV(card.querySelector("textarea").value);
      output.textContent = rows.length ? "Importing..." : "No rows found.";
      if (!rows.length) return;
      const res = await fetch(card.dataset.import, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rows })
      });
      output.textContent = JSON.stringify(await res.json(), null, 2);
    });
  });
})();
