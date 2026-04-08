function openOverlay(targetId) {
  const overlay = document.createElement('div');
  overlay.classList.add('overlay');
  document.body.appendChild(overlay);

  // скрываем диораму и параллакс-блоки
  document.querySelectorAll('.diorama, .parallax-block').forEach(el => el.style.display = 'none');

  // показываем модалку после затемнения
  setTimeout(() => {
    showView(targetId);

    // если открывается чат — проверяем авторизацию и подключаем сокет
    if (targetId === "view-chat") {
      if (document.body.dataset.isauth === "true") {
        connectChat();
      } else {
        const chatBox = document.getElementById("chat-box");
        const chatSendBtn = document.getElementById("chat-send-btn");
        const chatInput = document.getElementById("chat-input");

        if (chatBox) {
          chatBox.innerHTML = "<p>Для входа в чат нужна авторизация.</p>";
        }
        if (chatSendBtn) chatSendBtn.disabled = true;
        if (chatInput) chatInput.disabled = true;
      }
    }

    // запускаем fadeOut для overlay
    overlay.classList.add('fadeOut');
    overlay.addEventListener('animationend', () => {
      overlay.remove();
    });
  }, 300);
}

document.querySelectorAll('.open-modal').forEach(btn => {
  btn.addEventListener('click', () => {
    const target = btn.dataset.target;
    openOverlay(target);
  });
});

function showView(viewId) {
  document.querySelectorAll('.view.modal').forEach(v => v.style.display = 'none');
  const view = document.getElementById(viewId);
  if (view) {
    view.style.display = 'flex';

    // навешиваем обработчик на кнопку "Назад"
    const backBtn = view.querySelector('.modal-back-btn');
    if (backBtn) {
      backBtn.onclick = () => {
        view.style.display = 'none';
        // возвращаемся на домашнюю страницу
        document.querySelectorAll('.diorama, .parallax-block').forEach(el => el.style.display = '');
        location.hash = "home";
      };
    }
  }
}
