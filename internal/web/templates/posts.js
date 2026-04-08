const postsList = document.getElementById("posts-list");
const createPostBtn = document.getElementById("create-post-btn");
const postForm = document.getElementById("post-form");
const cancelPostBtn = document.getElementById("cancel-post-btn");
const sortSelect = document.getElementById("sort-select");
const clearBtn = document.getElementById("clear-sort-btn");

// создаём кнопку "Наверх"
const scrollTopBtn = document.createElement("button");
scrollTopBtn.id = "scroll-top-btn";
scrollTopBtn.textContent = "⬆ Наверх";
document.body.appendChild(scrollTopBtn);

// находим контейнер доски
const board = document.querySelector(".posts-board");

let suppressScrollTopBtn = false;

// следим за прокруткой именно доски
if (board) {
  board.addEventListener("scroll", () => {
    if (!suppressScrollTopBtn && board.scrollTop > 300) {
      scrollTopBtn.style.display = "block";
    } else {
      scrollTopBtn.style.display = "none";
    }
  });

  // при клике прокручиваем доску вверх
  scrollTopBtn.addEventListener("click", () => {
    board.scrollTo({ top: 0, behavior: "smooth" });
  });
}

// глобальная переменная для запоминания источника вызова формы
let formOrigin = null;

// универсальная функция обновления UI
function updateAuthUI() {
  if (document.body.dataset.isauth === "true") {
    createPostBtn.style.display = "inline-block";
  } else {
    createPostBtn.style.display = "none";
  }
}

// вызов при загрузке
updateAuthUI();

if (sortSelect && clearBtn) {
  sortSelect.addEventListener("change", async () => {
    const value = sortSelect.value;

    if (value === "mine") {
      const response = await fetch("/posts", { credentials: "include" });
      let posts = await response.json();
      const myUserId = document.body.dataset.userid;
      posts = posts.filter(p => p.user_id == myUserId);
      renderPosts(posts);
    } else {
      let url = "/posts";
      if (value === "rating_desc") url += "?sort=rating_desc";
      if (value === "rating_asc") url += "?sort=rating_asc";
      const response = await fetch(url, { credentials: "include" });
      const posts = await response.json();
      renderPosts(posts);
    }

    clearBtn.style.display = value ? "inline-block" : "none";
  });

  clearBtn.addEventListener("click", async () => {
    sortSelect.value = "";
    clearBtn.style.display = "none";
    const response = await fetch("/posts", { credentials: "include" });
    const posts = await response.json();
    renderPosts(posts);
  });
}

// загрузка постов
async function loadPosts() {
  try {
    const response = await fetch("/posts", { credentials: "include" });
    if (!response.ok) throw new Error("Ошибка загрузки постов");
    const posts = await response.json();
    renderPosts(posts);
  } catch (err) {
    console.error(err);
    postsList.innerHTML = "<p>Не удалось загрузить посты</p>";
  }
}

function renderPosts(posts) {
  postsList.innerHTML = "";
  posts.forEach(function(post) {
    const div = document.createElement("div");
    div.className = "post-item";

    let categoriesHtml = "";
    if (post.categories && post.categories.length > 0) {
      categoriesHtml = `<p class="categories">Категории: ${post.categories.join(", ")}</p>`;
    }

    let tagsHtml = "";
    if (post.tags && post.tags.length > 0) {
      tagsHtml = `<p class="tags">Теги: ${post.tags.join(", ")}</p>`;
    }

    let html = `
      <h3>${post.title}</h3>
      <p>${post.content}</p>
      ${categoriesHtml}
      ${tagsHtml}
      <p class="meta">
        Автор: ${post.username || "?"}, 
        Рейтинг: ${post.rating || 0}, 
        👍 ${post.likes || 0}, 👎 ${post.dislikes || 0}
      </p>
      `;

    if (document.body.dataset.isauth === "true" &&
        document.body.dataset.userid == post.user_id) {
      html += `
        <button class="edit-post-btn" data-id="${post.id}">Редактировать</button>
        <button class="delete-post-btn" data-id="${post.id}">Удалить</button>
      `;
    }

    html += `
      <button class="vote-btn" data-id="${post.id}" data-value="1">👍</button>
      <button class="vote-btn" data-id="${post.id}" data-value="-1">👎</button>
    `;

    div.innerHTML = html;

    div.addEventListener("click", async function() {
      const response = await fetch(`/posts?posts-id=${post.id}`, { credentials: "include" });
      if (response.ok) {
        const fullPost = await response.json();
        renderSinglePost(fullPost);
      }
    });

    postsList.appendChild(div);
  });

  document.querySelectorAll(".vote-btn").forEach(function(btn) {
    btn.addEventListener("click", async function(e) {
      e.stopPropagation();
      const postID = btn.dataset.id;
      const value = btn.dataset.value;
      await fetch("/vote", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ post_id: postID, value }),
        credentials: "include"
      });
      await loadPosts();
    });
  });

  document.querySelectorAll(".edit-post-btn").forEach(function(btn) {
    btn.addEventListener("click", function(e) {
      e.stopPropagation();
      openEditForm(btn.dataset.id);
    });
  });
  document.querySelectorAll(".delete-post-btn").forEach(function(btn) {
    btn.addEventListener("click", function(e) {
      e.stopPropagation();
      deletePost(btn.dataset.id);
    });
  });
}

if (createPostBtn) {
  createPostBtn.addEventListener("click", function() {
    formOrigin = "create"; // запомнили, что форма вызвана для создания
    postForm.style.display = "block";
    postForm.reset();
    document.getElementById("post-id").value = "";

    // временно отключаем кнопку "Наверх"
    suppressScrollTopBtn = true;
    postForm.scrollIntoView({ behavior: "smooth", block: "start" });
    setTimeout(() => suppressScrollTopBtn = false, 1000);
  });
}

if (cancelPostBtn) {
  console.log("Cancel button found, adding listener");
  cancelPostBtn.addEventListener("click", function() {
    console.log("Cancel clicked, formOrigin:", formOrigin);

    if (formOrigin === "create") {
      const board = document.querySelector(".posts-board");
      if (board) {
        console.log("Scrolling posts-board to top");
        board.scrollTo({ top: 0, behavior: "smooth" });
      }
    } else if (formOrigin) {
      const postElement = document.querySelector(`.edit-post-btn[data-id="${formOrigin}"]`);
      if (postElement) {
        console.log("Scrolling back to post", formOrigin);
        postElement.closest(".post-item")
                   .scrollIntoView({ behavior: "smooth", block: "start" });
      }
    }

    // закрываем форму и сбрасываем состояние чуть позже
    setTimeout(() => {
      postForm.style.display = "none";
      formOrigin = null;
    }, 500); // задержка 300 мс
  });
} else {
  console.log("Cancel button NOT found");
}

function openEditForm(postID) {
  fetch(`/posts?posts-id=${postID}`, { credentials: "include" })
    .then(res => res.json())
    .then(post => {
      formOrigin = post.id; // запомнили id поста
      postForm.style.display = "block";
      document.getElementById("post-id").value = post.id;
      document.getElementById("post-title").value = post.title;
      document.getElementById("post-content").value = post.content;

      const categoriesSelect = document.getElementById("categories");
      let postCategories = [];
      if (Array.isArray(post.categories)) {
        postCategories = post.categories;
      } else if (typeof post.categories === "string") {
        postCategories = post.categories.split(",").map(s => s.trim());
      }
      for (let option of categoriesSelect.options) {
        option.selected = postCategories.includes(option.text);
      }

      // временно отключаем кнопку "Наверх"
      suppressScrollTopBtn = true;
      setTimeout(() => {
        postForm.scrollIntoView({ behavior: "smooth", block: "start" });
        setTimeout(() => suppressScrollTopBtn = false, 1000);
      }, 50);
    });
}

async function deletePost(postID) {
  if (!confirm("Удалить пост?")) return;
  try {
    const response = await fetch("/delete-post", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({ id: postID }),
      credentials: "include"
    });
    if (response.ok) {
      await loadPosts();
    } else {
      alert("Ошибка удаления поста");
    }
  } catch (err) {
    console.error(err);
    alert("Ошибка соединения с сервером");
  }
}

// начальный рендер
loadPosts();

if (sortSelect) {
  sortSelect.value = "";
  clearBtn.style.display = "none";
}
