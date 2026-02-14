document.addEventListener('DOMContentLoaded', function() {
	document.addEventListener('submit', function(e) {
		var btn = e.target.querySelector('.htmx-submit-btn');
		if (btn) {
			e.target.classList.add('htmx-request');
		}
	});
});
