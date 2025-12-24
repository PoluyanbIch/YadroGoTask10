const API_BASE = 'http://localhost:28080/api';
const AUTH_BASE = 'http://localhost:28080/auth';

// --- 1. API Wrapper ---
async function apiRequest(url, method = 'GET', body = null, isJson = true) {
    const options = {
        method,
        credentials: 'include', // ВАЖНО: отправлять куки на бэкенд
        headers: {}
    };

    if (body) {
        if (isJson) {
            options.headers['Content-Type'] = 'application/json';
            options.body = JSON.stringify(body);
        } else {
            // Для Login/Register (x-www-form-urlencoded)
            options.headers['Content-Type'] = 'application/x-www-form-urlencoded';
            options.body = body;
        }
    }

    try {
        const response = await fetch(url, options);
        
        if (response.status === 401 || response.status === 403) {
            // Если токен протух — удаляем юзера из локального хранилища
            localStorage.removeItem('user');
            updateNav();
            // Если мы не на логине и это не опциональный запрос — редирект
            if (!window.location.pathname.includes('login') && !window.location.pathname.includes('index')) {
               // window.location.href = 'login.html'; // Можно раскомментировать для жесткого редиректа
            }
            throw new Error('Unauthorized');
        }

        const contentType = response.headers.get("content-type");
        if (contentType && contentType.includes("application/json")) {
            const data = await response.json();
            if (!response.ok) throw new Error(data.error || 'Error');
            return data;
        }
        return await response.text();
    } catch (err) {
        console.error("API Error:", err);
        throw err;
    }
}

// --- 2. Auth Helpers ---
function getUser() {
    return JSON.parse(localStorage.getItem('user'));
}

function updateNav() {
    const nav = document.getElementById('navbar');
    const user = getUser();
    
    let links = `<a href="index.html" class="brand">ComicSearch</a>`;
    
    if (user) {
        if (user.is_admin) {
            links += `<a href="admin.html" style="color: #ffc107;">Admin Panel</a>`;
        }
        links += `
            <div style="margin-left: auto;">
                <a href="profile.html">Profile (${user.login})</a>
                <a href="#" onclick="logout()">Logout</a>
            </div>
        `;
    } else {
        links += `
            <div style="margin-left: auto;">
                <a href="login.html">Login</a>
                <a href="register.html">Register</a>
            </div>
        `;
    }
    nav.innerHTML = links;
}

async function logout() {
    await apiRequest(`${AUTH_BASE}/logout`, 'POST');
    localStorage.removeItem('user');
    window.location.href = 'index.html';
}

// --- 3. UI Helpers ---
function renderComics(comics, containerId) {
    const container = document.getElementById(containerId);
    container.innerHTML = '';
    if (!comics || comics.length === 0) {
        container.innerHTML = '<p>No comics found.</p>';
        return;
    }
    comics.forEach(c => {
        // У комикса может быть поле url или imgUrl, и id
        // Если прилетел просто ID (из recent views), нужно это обработать, но пока считаем что прилетает объект
        const div = document.createElement('div');
        div.className = 'card';
        div.innerHTML = `
            <img src="${c.url || 'https://via.placeholder.com/200?text=Comic'}" alt="Comic">
            <h3>Comic #${c.id}</h3>
        `;
        div.onclick = () => window.location.href = `comic.html?id=${c.id}`;
        container.appendChild(div);
    });
}

// Запуск при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    updateNav();
    // Логика роутинга для конкретных страниц вызывается внутри самих HTML через <script>
});