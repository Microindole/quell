// 核心模块：state、init、导航、主题、窗口控制、toast/log

// 记录最后一次点击位置，用于主题波浪展开原点
let _themeX = '50%', _themeY = '50%';
document.addEventListener('mousedown', (e) => {
    _themeX = (e.clientX / window.innerWidth  * 100).toFixed(1) + '%';
    _themeY = (e.clientY / window.innerHeight * 100).toFixed(1) + '%';
});

const app = {
    state: {
        config: {},
        localTasks: [],
        remoteVideos: [],
        users: [],
        currentTab: 'local',
        theme: localStorage.getItem('theme') || 'system',
        downloadingParams: new Set(),
        selectedRemote: new Map(),
        downloadQueue: [],
        batchDownloading: false,
        currentUID: '',
        currentPage: 1,
        remoteTotal: 0,
        remoteTotalPages: 0,
        remotePageSize: 30,
        remoteSearchMode: 'user',
        remoteKeyword: '',
        remoteSort: 'pubdate',
        remoteSearchTimer: null,
        remoteSearching: false,
        dynamicUID: '',
        dynamicOffset: '',
        dynamicHasMore: false,
        dynamicLoading: false,
        dynamicCooldownUntil: 0,
        dynamics: [],
        pageSelectBvid: '',
        pageSelectTitle: '',
        pageSelectLength: '',
        pageSelectPages: [],
        pageStreamOptions: [],
        downloadPref: {
            quality_id: 0,
            codec: ''
        },
        loginPollingTimer: null,
        loginQrcodeKey: '',
        loginURL: ''
    },

    init() {
        this.loadConfig();
        this.setupEvents();
        this.setupTheme();
        this.setupResizeHandles();
        if (this.onRemoteSearchTypeChanged) this.onRemoteSearchTypeChanged();
        this.scanLocal();
    },

    // --- 导航 ---
    switchTab(tabName) {
        document.querySelectorAll('.view-section').forEach(el => el.classList.remove('active'));
        document.getElementById(`view-${tabName}`).classList.add('active');

        document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
        const btnIndex = ['local', 'remote', 'dynamics', 'logs', 'settings'].indexOf(tabName);
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
        if (document.startViewTransition) {
            document.documentElement.style.setProperty('--theme-x', _themeX);
            document.documentElement.style.setProperty('--theme-y', _themeY);
            document.startViewTransition(() => this.applyTheme(theme));
        } else {
            this.applyTheme(theme);
        }
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

    setupResizeHandles() {
        const handles = document.querySelectorAll('.resize-handle');
        if (!handles || handles.length === 0) return;

        let state = null;
        let ticking = false;
        const clearResizeState = () => {
            state = null;
            document.removeEventListener('pointermove', onPointerMove, true);
            document.removeEventListener('pointerup', onPointerUp, true);
            document.removeEventListener('pointercancel', onPointerUp, true);
            document.removeEventListener('mousemove', onMouseMoveFallback, true);
            document.removeEventListener('mouseup', onPointerUp, true);
        };

        const applyResize = () => {
            ticking = false;
            if (!state) return;
            const dx = state.lastMouseX - state.startMouseX;
            const dy = state.lastMouseY - state.startMouseY;

            let x = state.startX;
            let y = state.startY;
            let w = state.startW;
            let h = state.startH;
            const edge = state.edge;
            const minW = 900;
            const minH = 600;

            if (edge.includes('e')) w = state.startW + dx;
            if (edge.includes('s')) h = state.startH + dy;
            if (edge.includes('w')) {
                w = state.startW - dx;
                x = state.startX + dx;
            }
            if (edge.includes('n')) {
                h = state.startH - dy;
                y = state.startY + dy;
            }

            if (w < minW) {
                if (edge.includes('w')) x -= (minW - w);
                w = minW;
            }
            if (h < minH) {
                if (edge.includes('n')) y -= (minH - h);
                h = minH;
            }

            go.main.App.SetWindowBounds(Math.round(x), Math.round(y), Math.round(w), Math.round(h)).catch(() => {});
        };

        const onPointerMove = (e) => {
            if (!state) return;
            if (typeof e.buttons === 'number' && e.buttons === 0) {
                clearResizeState();
                return;
            }
            state.lastMouseX = e.screenX;
            state.lastMouseY = e.screenY;
            if (!ticking) {
                ticking = true;
                requestAnimationFrame(applyResize);
            }
        };

        // Fallback for environments where pointer events are incomplete.
        const onMouseMoveFallback = (e) => {
            if (!state) return;
            if (typeof e.buttons === 'number' && e.buttons === 0) {
                clearResizeState();
                return;
            }
            state.lastMouseX = e.screenX;
            state.lastMouseY = e.screenY;
            if (!ticking) {
                ticking = true;
                requestAnimationFrame(applyResize);
            }
        };

        const onPointerUp = () => clearResizeState();

        window.addEventListener('blur', clearResizeState, true);
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) clearResizeState();
        });
        window.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') clearResizeState();
        }, true);

        handles.forEach((handle) => {
            handle.addEventListener('pointerdown', async (e) => {
                if (e.button !== 0) return;
                if (window.runtime && window.runtime.WindowIsMaximised) {
                    try {
                        const isMax = await window.runtime.WindowIsMaximised();
                        if (isMax) return;
                    } catch (_) {}
                }
                e.preventDefault();
                e.stopPropagation();

                const bounds = await go.main.App.GetWindowBounds();
                state = {
                    edge: handle.dataset.edge || '',
                    startX: bounds.x,
                    startY: bounds.y,
                    startW: bounds.w,
                    startH: bounds.h,
                    startMouseX: e.screenX,
                    startMouseY: e.screenY,
                    lastMouseX: e.screenX,
                    lastMouseY: e.screenY
                };
                try {
                    if (handle.setPointerCapture && e.pointerId !== undefined) {
                        handle.setPointerCapture(e.pointerId);
                    }
                } catch (_) {}
                document.addEventListener('pointermove', onPointerMove, true);
                document.addEventListener('pointerup', onPointerUp, true);
                document.addEventListener('pointercancel', onPointerUp, true);
                document.addEventListener('mousemove', onMouseMoveFallback, true);
                document.addEventListener('mouseup', onPointerUp, true);
            });
        });
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
