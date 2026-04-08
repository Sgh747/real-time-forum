// static/js/dropdownlist.js
document.addEventListener("DOMContentLoaded", () => {
  const select = document.getElementById("categories");
  const chipsContainer = document.getElementById("selected-chips");
  const legacyBtn = document.getElementById("categoryDropdown");
  const legacyMenu = document.getElementById("categoryMenu");
  const legacyInput = document.getElementById("categoryInput");

  // Если есть <select multiple id="categories"> — используем новый режим
  if (select && chipsContainer) {
    // Рендер чипов для текущего выбора
    function renderChips() {
      chipsContainer.innerHTML = "";
      Array.from(select.selectedOptions).forEach(opt => {
        const chip = document.createElement("span");
        chip.className = "category-chip";
        chip.dataset.value = opt.value;
        chip.textContent = opt.textContent;

        const remove = document.createElement("button");
        remove.type = "button";
        remove.className = "chip-remove";
        remove.setAttribute("aria-label", "Удалить категорию");
        remove.textContent = "×";
        remove.addEventListener("click", () => {
          opt.selected = false;
          renderChips();
        });

        chip.appendChild(remove);
        chipsContainer.appendChild(chip);
      });
    }

    // При изменении select — обновляем чипы
    select.addEventListener("change", renderChips);

    // Инициалный рендер (если есть предвыбор)
    renderChips();

    // Удобство: клик по контейнеру фокусирует select
    chipsContainer.addEventListener("click", () => select.focus());
    return; // выходим — не инициализируем legacy dropdown
  }

  // --- Legacy dropdown mode (если вы ещё используете старый dropdown) ---
  // Если в шаблоне остался старый dropdown, поддерживаем его поведение.
  if (!legacyBtn || !legacyMenu || !legacyInput) return;

  // Открыть/закрыть меню
  legacyBtn.addEventListener("click", () => {
    const open = legacyMenu.classList.toggle("open");
    legacyBtn.setAttribute("aria-expanded", open ? "true" : "false");
  });

  // Выбор категории (одиночный выбор)
  legacyMenu.addEventListener("click", (e) => {
    const item = e.target.closest(".dropdown-item");
    if (!item) return;

    legacyBtn.textContent = item.textContent.trim() + " ▼";
    legacyInput.value = item.dataset.value;

    legacyMenu.classList.remove("open");
    legacyBtn.setAttribute("aria-expanded", "false");
  });

  // Закрыть при клике вне
  document.addEventListener("click", (e) => {
    if (!legacyBtn.contains(e.target) && !legacyMenu.contains(e.target)) {
      legacyMenu.classList.remove("open");
      legacyBtn.setAttribute("aria-expanded", "false");
    }
  });

  // Поддержка Enter
  legacyMenu.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      const item = document.activeElement.closest(".dropdown-item");
      if (item) {
        item.click();
        e.preventDefault();
      }
    }
  });

  // Валидация перед отправкой формы (если есть форма)
  const form = document.querySelector(".post-card");
  if (form) {
    form.addEventListener("submit", (e) => {
      if (!legacyInput.value) {
        e.preventDefault();
        alert("Пожалуйста, выберите категорию перед публикацией!");
      }
    });
  }
});
