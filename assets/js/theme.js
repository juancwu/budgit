// Apply saved theme or system preference on load
if (localStorage.theme === 'dark' || (!localStorage.theme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
	document.documentElement.classList.add('dark');
}

// Theme toggle handler
document.addEventListener('click', (e) => {
	if (e.target.closest('[data-theme-switcher]')) {
		e.preventDefault();
		const isDark = document.documentElement.classList.toggle('dark');
		localStorage.theme = isDark ? 'dark' : 'light';
	}
});
