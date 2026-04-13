const loginForm = document.getElementById("login-form");
const registerForm = document.getElementById("register-form");

// обработка логина
if (loginForm) {
  loginForm.addEventListener("submit", async function (e) {
    e.preventDefault();
    const email = loginForm.email.value;
    const password = loginForm.password.value;

    try {
      const response = await fetch("/login", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ email, password }),
        credentials: "include"
      });

      const data = await response.json();

      if (response.ok && data.success) {
        showToast("Успешный вход!", "success");
        document.body.dataset.isauth = "true";
        document.body.dataset.isregistered = "true";
        document.body.dataset.username = data.username;
        document.body.dataset.userid = data.userId;

        loginForm.reset();

        updateUI();
        updateAuthUI();
        connectChat();
        await loadPosts();

        // после входа → на главную
        location.hash = "#home";
        window.scrollTo({ top: 0, behavior: "smooth" });
      } else {
        showToast("Ошибка входа: " + data.error, "error");
      }
    } catch (err) {
      console.error("Ошибка запроса:", err);
      showToast("Ошибка соединения с сервером", "error");
    }
  });
}

// обработка регистрации
if (registerForm) {
  registerForm.addEventListener("submit", async function (e) {
    e.preventDefault();
    const dataForm = {
      username: registerForm.username.value,
      email: registerForm.email.value,
      password: registerForm.password.value,
      first_name: registerForm.first_name.value,
      last_name: registerForm.last_name.value,
      age: registerForm.age.value,
      gender: registerForm.gender.value,
    };

    try {
      const response = await fetch("/register", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams(dataForm),
        credentials: "include"
      });

      const data = await response.json();

      if (response.ok && data.success) {
        showToast("Регистрация успешна! Теперь войдите.", "success");
        document.body.dataset.isregistered = "true";
        document.body.dataset.isauth = "false";
        document.body.dataset.username = "";
        document.body.dataset.userid = "";

        registerForm.reset();
        updateUI();
        updateAuthUI();

        // ФИКС: после регистрации → на главную страницу
        location.hash = "#home";
        window.scrollTo({ top: 0, behavior: "smooth" });
      } else {
        showToast("Ошибка регистрации: " + data.error, "error");
      }
    } catch (err) {
      console.error("Ошибка запроса:", err);
      showToast("Ошибка соединения с сервером", "error");
    }
  });
}

// обработка выхода
document.addEventListener("click", async function (e) {
  if (e.target && e.target.id === "logout-btn") {
    try {
      await fetch("/logout", { method: "POST", credentials: "include" });
      showToast("Вы вышли из аккаунта", "success");

      if (typeof window.closeSocket === "function") {
        window.closeSocket("logout");
      }

      document.body.dataset.isauth = "false";
      document.body.dataset.isregistered = "false";
      document.body.dataset.username = "";
      document.body.dataset.userid = "";

      updateUI();
      updateAuthUI();
      location.hash = "#home";
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (err) {
      console.error("Ошибка выхода:", err);
      showToast("Ошибка соединения с сервером", "error");
    }
  }
});

// синхронизация состояния авторизации
async function syncAuthState() {
  const nav = document.getElementById("navbar");
  nav.innerHTML = "<span>Загрузка...</span>";

  try {
    const response = await fetch("/me", { credentials: "include" });
    const data = await response.json();
    if (response.ok && data.success) {
      document.body.dataset.isauth = "true";
      document.body.dataset.isregistered = "true";
      document.body.dataset.username = data.username;
      document.body.dataset.userid = data.userId;
    } else {
      document.body.dataset.isauth = "false";
      document.body.dataset.isregistered = "false";
      document.body.dataset.username = "";
      document.body.dataset.userid = "";
    }
    updateUI();
    await loadPosts();
  } catch (err) {
    console.error("Ошибка проверки авторизации:", err);
  }
}
