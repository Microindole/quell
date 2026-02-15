// 核心模块：state、init、导航、主题、窗口控制、toast/log

const app = {
    state: {
        config: {},
        localTasks: [],
        remoteVideos: [],
        users: [],
        currentTab: 'local',
        theme: localStorage.getItem('theme') || 'system',
        downloadingParams: new Set(),
        currentUID: '',
        currentPage: 1,
        pageSelectBvid: '',
        pageSelectTitle: '',
        pageSelectPages: []
    },

    init() {
        this.loadConfig();
        this.setupEvents();
        this.setupTheme();
        this.scanLocal();
    },

    // --- 导航 ---
    switchTab(tabName) {
        document.querySelectorAll('.view-section').forEach(el => el.classList.remove('active'));
        document.getElementById(`view-${tabName}`).classList.add('active');

        document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
        const btnIndex = ['local', 'remote', 'logs', 'settings'].indexOf(tabName);
        if (btnIndex >= 0) {
            document.querySelectorAll('.nav-item')[btnIndex].classList.add('active');
        }

        this.state.currentTab = tabName;
    },

    // --- 主题 ---
    setupTheme() {
        this.applyTheme(this.state.theme);
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
            if (this.state.theme === 'system') this.applyTheme('system');
        });
    },

    setTheme(theme) {
        this.state.theme = theme;
        localStorage.setItem('theme', theme);
        this.applyTheme(theme);
    },

    applyTheme(theme) {
        const root = document.documentElement;
        if (theme === 'system') {
            root.setAttribute('data-theme', window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
        } else {
            root.setAttribute('data-theme', theme);
        }
        const sel = document.getElementById('cfgTheme');
        if (sel) sel.value = theme;
    },

    // --- 窗口控制 ---
    windowMinimise() { go.main.App.WindowMinimise(); },
    windowMaximise() { go.main.App.WindowMaximise(); },
    windowClose()    { go.main.App.WindowClose(); },

    // --- Toast 通知 ---
    toast(msg, type = 'info') {
        const container = document.getElementById('toastContainer');
        const el = document.createElement('div');
        el.className = `toast ${type}`;
        el.innerText = msg;
        container.appendChild(el);

        setTimeout(() => {
            el.style.opacity = '0';
            setTimeout(() => el.remove(), 300);
        }, 3000);

        this.log(`[${type.toUpperCase()}] ${msg}`);
    },

    // --- 日志 ---
    log(msg) {
        const container = document.getElementById('logContainer');
        if (!container) return;
        const entry = document.createElement('div');
        entry.className = 'log-entry';
        const time = new Date().toLocaleTimeString();
        entry.innerHTML = `<span class="log-time">${time}</span> ${msg}`;
        container.appendChild(entry);
        container.scrollTop = container.scrollHeight;
    },

    clearLogs() {
        const container = document.getElementById('logContainer');
        if (container) container.innerHTML = '';
    }
};

window.onload = () => {
    try {
        app.init();
    } catch (e) {
        document.body.innerHTML = `<div style="color:red; padding:20px;">
            <h1>Startup Error</h1>
            <pre>${e.message}\n${e.stack}</pre>
        </div>`;
    }
};
