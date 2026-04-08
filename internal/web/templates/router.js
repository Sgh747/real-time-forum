function showView(viewId) {
  document.querySelectorAll('.view').forEach(v => {
    if (v.id !== 'view-home') { // диораму не скрываем
      v.style.display = 'none';
    }
  });
  const view = document.getElementById(viewId);
  if (view) view.style.display = 'flex';
}

function routeHandler() {
  const route = location.hash.replace('#', '');
  const isAuth = document.body.dataset.isauth === "true";

  switch (route) {
    case 'register':
      document.getElementById('register-block').scrollIntoView({ behavior: 'smooth' });
      break;
    case 'login':
      if (isAuth) {
        // если авторизован, не пускаем на login, а возвращаем на home
        location.hash = '#home';
        showView('view-home');
      } else {
        document.getElementById('login-block').scrollIntoView({ behavior: 'smooth' });
      }
      break;
    case 'posts':
      document.getElementById('posts-block').scrollIntoView({ behavior: 'smooth' });
      break;
    case 'chat':
      document.getElementById('chat-block').scrollIntoView({ behavior: 'smooth' });
      break;
    case 'home':
      showView('view-home');
      break;
    default:
      showView('view-home');
  }
}

window.addEventListener('hashchange', routeHandler);

// при загрузке страницы
if (!location.hash) location.hash = '#home';
routeHandler();
