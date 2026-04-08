document.addEventListener("DOMContentLoaded", function() {
    console.log("credits.js загружен");

    const creditsButton = document.querySelector('button.open-modal[data-target="view-credits"]');
    const creditsAudio = document.getElementById("credits-audio");
    const backBtn = document.querySelector('#view-credits .modal-back-btn');
    const creditsModal = document.getElementById("view-credits");
    const slides = document.querySelectorAll(".credits-slide");
    let currentSlide = 0;
    let slideInterval = null;

    function showSlide(index) {
        slides.forEach((slide, i) => {
            slide.classList.toggle("active", i === index);
        });
    }

    function startSlideshow() {
        if (slideInterval) return; // уже запущен
        showSlide(0);
        currentSlide = 0;
        slideInterval = setInterval(() => {
            currentSlide = (currentSlide + 1) % slides.length;
            showSlide(currentSlide);
        }, 5000);
    }

    function stopSlideshow() {
        if (slideInterval) {
            clearInterval(slideInterval);
            slideInterval = null;
        }
    }

    if (creditsButton && creditsModal) {
        creditsButton.addEventListener("click", function() {
            console.log("Кнопка титров нажата");
            creditsModal.style.display = "flex";
            startSlideshow();

            // запуск аудио
            creditsAudio.currentTime = 0;
            creditsAudio.play().catch(err => {
                console.error("Ошибка воспроизведения аудио:", err);
            });
        });
    }

    if (backBtn && creditsAudio && creditsModal) {
        backBtn.addEventListener("click", function() {
            console.log("Кнопка Назад нажата");

            // остановить аудио
            creditsAudio.pause();
            creditsAudio.currentTime = 0;

            // остановить слайдшоу
            stopSlideshow();

            // скрыть модал
            creditsModal.style.display = "none";
        });
    }
});
