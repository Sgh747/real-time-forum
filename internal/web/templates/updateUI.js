function updateUI() {
  const nav = document.getElementById("navbar");
  const isAuth = document.body.dataset.isauth === "true";
  const isRegistered = document.body.dataset.isregistered === "true";
  const username = document.body.dataset.username;

  // Навигация
  if (isAuth) {
    nav.innerHTML = `
      <a href="#" class="nav-link" data-scroll="view-home">Главная</a>
      <a href="#" class="nav-link" data-scroll="login-block-user">Площадь</a>
      <a href="#" class="nav-link" data-scroll="posts-block">Посты</a>
      <a href="#" class="nav-link" data-scroll="chat-block">Чат</a>
      <a href="#" class="nav-link" data-scroll="register-block-logged-in">Выход (${username})</a>
      <a href="#" class="nav-link" data-scroll="credits-block">Титры</a>
    `;
    const createBtn = document.getElementById("create-post-btn");
    if (createBtn) createBtn.classList.remove("hidden");
  } else {
    nav.innerHTML = `
      <a href="#" class="nav-link" data-scroll="view-home">Главная</a>
      <a href="#" class="nav-link" data-scroll="register-block-guest">Регистрация</a>
      <a href="#" class="nav-link" data-scroll="login-block-guest">Вход</a>
      <a href="#" class="nav-link" data-scroll="posts-block">Посты</a>
      <a href="#" class="nav-link" data-scroll="chat-block">Чат</a>
      <a href="#" class="nav-link" data-scroll="credits-block">Титры</a>
    `;
    const createBtn = document.getElementById("create-post-btn");
    if (createBtn) createBtn.classList.add("hidden");
  }

  // Параллакс‑блоки
  const regGuest    = document.getElementById("register-block-guest");
  const regRegistered = document.getElementById("register-block-registered");
  const regLoggedIn = document.getElementById("register-block-logged-in");
  const loginGuest  = document.getElementById("login-block-guest");
  const loginUser   = document.getElementById("login-block-user");

  if (regGuest && regRegistered && regLoggedIn) {
    regGuest.classList.toggle("hidden", isRegistered || isAuth);
    regRegistered.classList.toggle("hidden", !isRegistered || isAuth);
    regLoggedIn.classList.toggle("hidden", !isAuth);
  }

  if (loginGuest && loginUser) {
    loginGuest.classList.toggle("hidden", isAuth);
    loginUser.classList.toggle("hidden", !isAuth);
  }

  // ФИКС: подсвечиваем активную ссылку в навбаре
  highlightActiveNav();
}

// Подсвечивает кнопку "Главная" когда пользователь наверху страницы
function highlightActiveNav() {
  const links = document.querySelectorAll(".nav-link[data-scroll]");

  // Определяем текущую секцию через IntersectionObserver или scrollY
  const sections = [
    "view-home", "register-block-guest", "register-block-registered",
    "login-block-guest", "login-block-user", "register-block-logged-in",
    "posts-block", "chat-block", "credits-block"
  ];

  // Снимаем active со всех
  links.forEach(l => l.classList.remove("nav-active"));

  // Находим какая секция сейчас в зоне видимости
  let activeSection = null;
  for (const id of sections) {
    const el = document.getElementById(id);
    if (!el) continue;
    const rect = el.getBoundingClientRect();
    if (rect.top <= window.innerHeight / 2 && rect.bottom >= 0) {
      activeSection = id;
      break;
    }
  }

  // Подсвечиваем нужную ссылку
  links.forEach(link => {
    if (link.dataset.scroll === activeSection) {
      link.classList.add("nav-active");
    }
  });
}

// Обновляем активную ссылку при прокрутке
window.addEventListener("scroll", highlightActiveNav, { passive: true });

// подтягиваем состояние при первой загрузке
document.addEventListener("DOMContentLoaded", async () => {
  await syncAuthState();
  updateUI();
  updateAuthUI();
  if (!location.hash || (document.body.dataset.isauth === "true" && location.hash === "#login")) {
    location.hash = "#home";
  }
  routeHandler();
});

// и при смене маршрута
window.addEventListener("hashchange", async () => {
  await syncAuthState();
  updateUI();
  updateAuthUI();
  await loadPosts();
});

// плавный скролл по data-scroll
document.addEventListener("click", (e) => {
  const link = e.target.closest("[data-scroll]");
  if (link) {
    e.preventDefault();
    const targetId = link.dataset.scroll;
    const target = document.getElementById(targetId);
    if (target) {
      target.scrollIntoView({ behavior: "smooth" });
    }
  }
});
