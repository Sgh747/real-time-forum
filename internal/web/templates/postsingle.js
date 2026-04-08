function renderSinglePost(post) {
  const modal = document.getElementById("post-modal");
  const modalContent = document.getElementById("modal-content");
  const modalTitle = document.getElementById("modal-title");

  modalTitle.textContent = post.title;

  modalContent.innerHTML = `
    <p>${post.content}</p>
    <p class="meta">Автор: ${post.username}, 👍 ${post.likes}, 👎 ${post.dislikes}</p>
    <button class="vote-btn" data-id="${post.id}" data-value="1">👍</button>
    <button class="vote-btn" data-id="${post.id}" data-value="-1">👎</button>

    <!-- Форма комментариев теперь выше -->
    <form id="comment-form">
      <input type="hidden" name="post_id" value="${post.id}">
      <textarea name="content" placeholder="Ваш комментарий"></textarea>
      <button type="submit">Отправить</button>
    </form>

    <!-- Комментарии ниже -->
    <div id="comments"></div>
  `;

  modal.style.display = "flex";

  // Кнопка назад
  modal.querySelector(".modal-back-btn").onclick = function() {
    modal.style.display = "none";
  };


  // Обработчики голосования за пост
  modalContent.querySelectorAll(".vote-btn").forEach(function(btn) {
    btn.addEventListener("click", async function() {
      const postID = btn.dataset.id;
      const value = btn.dataset.value;
      await fetch("/vote", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ post_id: postID, value }),
        credentials: "include"
      });
      const response = await fetch(`/posts?posts-id=${postID}`, { credentials: "include" });
      if (response.ok) {
        const updatedPost = await response.json();
        renderSinglePost(updatedPost);
      }
      // обновляем список постов
      await loadPosts();
    });
  });

  // Загружаем комментарии
  loadComments(post.id);

  // Отправка комментария
  const commentForm = document.getElementById("comment-form");
  commentForm.addEventListener("submit", async function(e) {
    e.preventDefault();
    const formData = new FormData(commentForm);
    const response = await fetch("/add-comment", {
      method: "POST",
      headers: { "Accept": "application/json" },
      body: new URLSearchParams(formData),
      credentials: "include"
    });
    if (response.ok) {
      loadComments(post.id);
      commentForm.reset();
    } else {
      alert("Ошибка добавления комментария");
    }
  });
}

function loadComments(postId) {
  const commentsDiv = document.getElementById("comments");
  fetch("/comments?post_id=" + postId, { credentials: "include" })
    .then(function(response) {
      if (response.ok) return response.json();
    })
    .then(function(comments) {
      if (comments) {
        commentsDiv.innerHTML = comments.map(function(c) {
          return `
            <p><b>${c.user}</b>: ${c.content}</p>
            <p>👍 ${c.likes} 👎 ${c.dislikes}</p>
            <button class="comment-vote-btn" data-id="${c.id}" data-value="1">👍</button>
            <button class="comment-vote-btn" data-id="${c.id}" data-value="-1">👎</button>
          `;
        }).join("");

        // обработчики голосования за комментарии
        document.querySelectorAll(".comment-vote-btn").forEach(function(btn) {
          btn.addEventListener("click", async function() {
            const commentID = btn.dataset.id;
            const value = btn.dataset.value;
            await fetch("/vote-comment", {
              method: "POST",
              headers: { "Content-Type": "application/x-www-form-urlencoded" },
              body: new URLSearchParams({ comment_id: commentID, value }),
              credentials: "include"
            });
            loadComments(postId); // обновляем список
          });
        });
      }
    });
}
