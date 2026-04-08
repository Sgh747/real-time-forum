document.addEventListener("DOMContentLoaded", () => {
  const blocks = document.querySelectorAll(".content-area");

  // Hover-логика
  blocks.forEach(block => {
    block.addEventListener("mouseenter", () => {
      block.classList.add("hover-active");
    });
    block.addEventListener("mouseleave", () => {
      block.classList.remove("hover-active");
    });
  });

  function updateOpacity() {
    const viewportCenter = window.innerHeight / 2;

    blocks.forEach(block => {
      if (block.classList.contains("hover-active")) return;

      const rect = block.getBoundingClientRect();
      const blockCenter = rect.top + rect.height / 2;

      const distance = Math.abs(blockCenter - viewportCenter);
      const maxDistance = viewportCenter;

      // Чем дальше от центра, тем меньше прозрачность
      let ratio = 1 - distance / maxDistance;
      ratio = Math.max(0, Math.min(1, ratio)); // ограничиваем 0–1

      const opacity = 0.1 + ratio * (0.9 - 0.1);
      block.style.opacity = opacity;
    });
  }

  // Обновляем при скролле и ресайзе
  window.addEventListener("scroll", updateOpacity);
  window.addEventListener("resize", updateOpacity);

  // Первичный вызов
  updateOpacity();
});
