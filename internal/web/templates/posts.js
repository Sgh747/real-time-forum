const postsList    = document.getElementById("posts-list");
const createPostBtn= document.getElementById("create-post-btn");
const postForm     = document.getElementById("post-form");
const cancelPostBtn= document.getElementById("cancel-post-btn");
const sortSelect   = document.getElementById("sort-select");
const clearBtn     = document.getElementById("clear-sort-btn");

// кнопка "Наверх"
const scrollTopBtn = document.createElement("button");
scrollTopBtn.id = "scroll-top-btn";
scrollTopBtn.textContent = "⬆ Наверх";
document.body.appendChild(scrollTopBtn);

const board = document.querySelector(".posts-board");
let suppressScrollTopBtn = false;

if (board) {
  board.addEventListener("scroll", () => {
    scrollTopBtn.style.display = (!suppressScrollTopBtn && board.scrollTop > 300) ? "block" : "none";
  });
  scrollTopBtn.addEventListener("click", () => board.scrollTo({ top: 0, behavior: "smooth" }));
}

let formOrigin = null;

function updateAuthUI() {
  if (createPostBtn) {
    createPostBtn.style.display = document.body.dataset.isauth === "true" ? "inline-block" : "none";
  }
}
updateAuthUI();

if (sortSelect && clearBtn) {
  sortSelect.addEventListener("change", async () => {
    const value = sortSelect.value;
    if (value === "mine") {
      const response = await fetch("/posts", { credentials: "include" });
      let posts = await response.json();
      const myUserId = document.body.dataset.userid;
      renderPosts(posts.filter(p => p.user_id == myUserId));
    } else {
      let url = "/posts";
      if (value === "rating_desc") url += "?sort=rating_desc";
      if (value === "rating_asc")  url += "?sort=rating_asc";
      const response = await fetch(url, { credentials: "include" });
      renderPosts(await response.json());
    }
    clearBtn.style.display = value ? "inline-block" : "none";
  });

  clearBtn.addEventListener("click", async () => {
    sortSelect.value = "";
    clearBtn.style.display = "none";
    renderPosts(await (await fetch("/posts", { credentials: "include" })).json());
  });
}

async function loadPosts() {
  try {
    const response = await fetch("/posts", { credentials: "include" });
    if (!response.ok) throw new Error("Ошибка загрузки постов");
    renderPosts(await response.json());
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
      </p>`;

    if (document.body.dataset.isauth === "true" && document.body.dataset.userid == post.user_id) {
      html += `
        <button class="edit-post-btn" data-id="${post.id}">Редактировать</button>
        <button class="delete-post-btn" data-id="${post.id}">Удалить</button>`;
    }

    html += `
      <button class="vote-btn" data-id="${post.id}" data-value="1">👍</button>
      <button class="vote-btn" data-id="${post.id}" data-value="-1">👎</button>`;

    div.innerHTML = html;

    div.addEventListener("click", async function() {
      const response = await fetch(`/posts?posts-id=${post.id}`, { credentials: "include" });
      if (response.ok) renderSinglePost(await response.json());
    });

    postsList.appendChild(div);
  });

  document.querySelectorAll(".vote-btn").forEach(function(btn) {
    btn.addEventListener("click", async function(e) {
      e.stopPropagation();
      await fetch("/vote", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ post_id: btn.dataset.id, value: btn.dataset.value }),
        credentials: "include"
      });
      await loadPosts();
    });
  });

  document.querySelectorAll(".edit-post-btn").forEach(btn => {
    btn.addEventListener("click", function(e) { e.stopPropagation(); openEditForm(btn.dataset.id); });
  });
  document.querySelectorAll(".delete-post-btn").forEach(btn => {
    btn.addEventListener("click", function(e) { e.stopPropagation(); deletePost(btn.dataset.id); });
  });
}

if (createPostBtn) {
  createPostBtn.addEventListener("click", function() {
    formOrigin = "create";
    postForm.style.display = "block";
    postForm.reset();
    document.getElementById("post-id").value = "";
    suppressScrollTopBtn = true;
    postForm.scrollIntoView({ behavior: "smooth", block: "start" });
    setTimeout(() => suppressScrollTopBtn = false, 1000);
  });
}

// ── ГЛАВНЫЙ ФИКС: перехватываем submit, отправляем через fetch ──────────────
if (postForm) {
  postForm.addEventListener("submit", async function(e) {
    e.preventDefault(); // не даём форме перезагрузить страницу

    const postId  = document.getElementById("post-id").value;
    const title   = document.getElementById("post-title").value.trim();
    const content = document.getElementById("post-content").value.trim();

    if (!title || !content) {
      showToast("Заполните заголовок и содержание", "error");
      return;
    }

    const categoriesSelect = document.getElementById("categories");
    const selectedCategories = Array.from(categoriesSelect.selectedOptions).map(o => o.value);

    if (selectedCategories.length === 0) {
      showToast("Выберите хотя бы одну категорию", "error");
      return;
    }

    const body = new URLSearchParams({ title, content });
    selectedCategories.forEach(c => body.append("categories", c));

    const isEdit = postId !== "";
    if (isEdit) body.append("id", postId);

    const url = isEdit ? "/edit-post" : "/create-post";

    try {
      const response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body,
        credentials: "include"
      });

      if (response.ok) {
        showToast(isEdit ? "Пост обновлён!" : "Пост создан!", "success");
        postForm.style.display = "none";
        postForm.reset();
        formOrigin = null;
        await loadPosts();
      } else {
        const text = await response.text();
        showToast("Ошибка: " + text, "error");
      }
    } catch (err) {
      console.error(err);
      showToast("Ошибка соединения с сервером", "error");
    }
  });
}

if (cancelPostBtn) {
  cancelPostBtn.addEventListener("click", function() {
    if (formOrigin === "create") {
      if (board) board.scrollTo({ top: 0, behavior: "smooth" });
    } else if (formOrigin) {
      const postElement = document.querySelector(`.edit-post-btn[data-id="${formOrigin}"]`);
      if (postElement) postElement.closest(".post-item").scrollIntoView({ behavior: "smooth", block: "start" });
    }
    setTimeout(() => {
      postForm.style.display = "none";
      formOrigin = null;
    }, 500);
  });
}

function openEditForm(postID) {
  fetch(`/posts?posts-id=${postID}`, { credentials: "include" })
    .then(res => res.json())
    .then(post => {
      formOrigin = post.id;
      postForm.style.display = "block";
      document.getElementById("post-id").value    = post.id;
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
        option.selected = postCategories.includes(option.text) || postCategories.includes(option.value);
      }

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
      showToast("Пост удалён", "success");
      await loadPosts();
    } else {
      showToast("Ошибка удаления поста", "error");
    }
  } catch (err) {
    console.error(err);
    showToast("Ошибка соединения с сервером", "error");
  }
}

loadPosts();

if (sortSelect) {
  sortSelect.value = "";
  clearBtn.style.display = "none";
}
