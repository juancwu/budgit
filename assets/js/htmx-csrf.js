document.addEventListener('DOMContentLoaded', function() {
	// Listen for htmx requests and add CSRF token header
	document.body.addEventListener('htmx:configRequest', function(event) {
		// Get CSRF token from meta tag
		const meta = document.querySelector('meta[name="csrf-token"]');
		if (meta) {
			// Add token as X-CSRF-Token header to all HTMX requests
			event.detail.headers['X-CSRF-Token'] = meta.getAttribute('content');
		}
	});
});
