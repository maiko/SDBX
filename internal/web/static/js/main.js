// SDBX Web UI - Shared Utilities

// --- Cookie Helper ---

function getCookie(name) {
    var match = document.cookie.match('(^|;)\\s*' + name + '=([^;]+)');
    return match ? match[2] : '';
}

// --- CSRF-aware Fetch ---

function csrfFetch(url, options) {
    options = options || {};
    options.headers = options.headers || {};
    var token = getCookie('csrf_token');
    if (token) {
        options.headers['X-CSRF-Token'] = token;
    }
    return fetch(url, options);
}

// --- Toast Notifications ---

function showToast(message, type) {
    type = type || 'success';
    var container = document.getElementById('toast-container');
    if (!container) return;

    var toast = document.createElement('div');
    toast.className = 'toast ' + type;

    var icon = type === 'success' ? '\u2713' : '\u2717';

    var iconSpan = document.createElement('span');
    iconSpan.className = 'toast-icon';
    iconSpan.textContent = icon;

    var msgSpan = document.createElement('span');
    msgSpan.className = 'toast-message';
    msgSpan.textContent = message;

    toast.appendChild(iconSpan);
    toast.appendChild(msgSpan);
    container.appendChild(toast);

    setTimeout(function() {
        toast.style.animation = 'slideIn 0.3s ease reverse';
        setTimeout(function() { toast.remove(); }, 300);
    }, 4000);
}

// --- Active Nav Link ---

document.addEventListener('DOMContentLoaded', function() {
    var path = window.location.pathname;
    document.querySelectorAll('.nav-link').forEach(function(link) {
        var href = link.getAttribute('href');
        if (href === path || (path.startsWith(href) && href !== '/')) {
            link.classList.add('active');
        } else if (href === '/' && path === '/') {
            link.classList.add('active');
        }
    });

    // Apply saved theme
    var savedTheme = localStorage.getItem('sdbx-theme');
    if (savedTheme) {
        document.documentElement.setAttribute('data-theme', savedTheme);
    }
});

// --- Dark Mode Toggle ---

function toggleTheme() {
    var current = document.documentElement.getAttribute('data-theme');
    var next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('sdbx-theme', next);
}
