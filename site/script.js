document.addEventListener('DOMContentLoaded', () => {
    // Demo Switcher Logic
    const switcherBtns = document.querySelectorAll('.demo-switcher button');
    const previews = document.querySelectorAll('.preview-item');

    switcherBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const target = btn.getAttribute('data-target');

            // Toggle Button Active State
            switcherBtns.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');

            // Toggle Preview Visibility
            previews.forEach(p => {
                p.classList.remove('active');
                if (p.id === `${target}-preview`) {
                    p.classList.add('active');
                }
            });
        });
    });

    // Simple Scroll Reveal Effect
    const observerOptions = {
        threshold: 0.1
    };

    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);

    // Apply reveal styles to sections
    const revealElements = document.querySelectorAll('.feature-card, .section-title, .download-card');
    revealElements.forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(30px)';
        el.style.transition = 'all 0.6s ease-out';
        observer.observe(el);
    });

    // Log welcome message
    console.log("%c Quell Website Loaded! ", "background: #f82662; color: #fff; font-weight: bold;");
});
