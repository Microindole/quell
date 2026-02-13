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
        currentPage: 1
    },

    init() {
        this.loadConfig();
        this.setupEvents();
        this.setupTheme();
        this.scanLocal();
    },

    // --- Navigation ---
    switchTab(tabName) {
        document.querySelectorAll('.view-section').forEach(el => el.classList.remove('active'));
        document.getElementById(`view-${tabName}`).classList.add('active');

        document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
        const btnIndex = ['local', 'remote', 'logs', 'settings'].indexOf(tabName);
        if (btnIndex >= 0 && document.querySelectorAll('.nav-item')[btnIndex]) {
            document.querySelectorAll('.nav-item')[btnIndex].classList.add('active');
        }

        this.state.currentTab = tabName;
    },

    // --- Theme ---
    setupTheme() {
        this.applyTheme(this.state.theme);
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
            if (this.state.theme === 'system') {
                this.applyTheme('system');
            }
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
            const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            root.setAttribute('data-theme', isDark ? 'dark' : 'light');
        } else {
            root.setAttribute('data-theme', theme);
        }

        const sel = document.getElementById('cfgTheme');
        if (sel) sel.value = theme;
    },

    // --- Window Controls ---
    windowMinimise() {
        go.main.App.WindowMinimise();
    },

    windowMaximise() {
        go.main.App.WindowMaximise();
    },

    windowClose() {
        go.main.App.WindowClose();
    },

    // --- Wails Events (替代 SSE) ---
    setupEvents() {
        const statusEl = document.getElementById('statusBar');

        window.runtime.EventsOn("merge", (data) => {
            if (data.status === 'processing') {
                this.toast(`正在合并: ${data.title}`, 'success');
            } else if (data.status === 'done') {
                const outPath = data.output || "合并完成";
                this.toast(`合并完成: ${outPath}`, 'success');
                this.log(`[SUCCESS] 合并完成，文件位于: ${outPath}`);

                if (data.index >= 0 && data.index < this.state.localTasks.length) {
                    this.state.localTasks[data.index].Status = 'SUCCESS';
                    this.renderLocalList();
                }
            } else if (data.status === 'error') {
                this.toast(`合并失败: ${data.error}`, 'error');
                if (data.index >= 0 && data.index < this.state.localTasks.length) {
                    this.state.localTasks[data.index].Status = '失败';
                    this.renderLocalList();
                }
            }
        });

        window.runtime.EventsOn("download", (data) => {
            if (data.status === 'done') {
                this.toast(`下载完成: ${data.bvid}`, 'success');
                this.state.downloadingParams.delete(data.bvid);
                this.renderRemoteList();
            } else if (data.status === 'error') {
                this.toast(`下载失败[${data.bvid}]: ${data.error}`, 'error');
                this.state.downloadingParams.delete(data.bvid);
                this.renderRemoteList();
            }
        });

        window.runtime.EventsOn("progress", (data) => {
            if (statusEl) {
                statusEl.innerText = `[${data.bvid}] ${data.message}`;
            }
        });
    },

    // --- API (Wails Go Bindings) ---
    async loadConfig() {
        try {
            const cfg = await go.main.App.GetConfig();
            this.state.config = cfg;

            document.getElementById('cfgBiliDir').value = cfg.BiliDir || '';
            document.getElementById('cfgFFmpeg').value = cfg.FFmpegPath || '';
            document.getElementById('cfgSessdata').value = cfg.SESSDATA || '';
        } catch (e) {
            this.toast('无法加载配置: ' + e, 'error');
        }
    },

    async saveConfig() {
        const payload = {
            BiliDir: document.getElementById('cfgBiliDir').value,
            FFmpegPath: document.getElementById('cfgFFmpeg').value,
            SESSDATA: document.getElementById('cfgSessdata').value
        };
        try {
            const err = await go.main.App.SaveConfig(payload);
            if (err) {
                throw new Error(err);
            }
            this.state.config = payload;
            this.toast('配置已保存', 'success');
        } catch (e) {
            this.toast('保存配置失败: ' + e, 'error');
        }
    },

    async scanLocal() {
        const listEl = document.getElementById('localVideoList');

        try {
            const tasks = await go.main.App.ScanVideos();
            this.state.localTasks = tasks || [];
            this.renderLocalList();
            this.toast(`扫描完成，找到 ${this.state.localTasks.length} 个视频`, 'success');
        } catch (e) {
            listEl.innerHTML = `<div class="empty-state">扫描失败: ${e}</div>`;
            this.toast('扫描失败: ' + e, 'error');
        }
    },

    triggerMerge(index) {
        go.main.App.MergeVideo(index);
        this.toast('已开始合并任务', 'success');
        this.state.localTasks[index].Status = '处理中...';
        this.renderLocalList();
    },

    // --- Remote Search ---
    async searchRemote() {
        const keyword = document.getElementById('searchInput').value.trim();
        if (!keyword) return;

        const userResultsEl = document.getElementById('userResults');
        const videoListEl = document.getElementById('remoteVideoList');
        const videoHeaderEl = document.getElementById('videoListHeader');

        userResultsEl.style.display = 'none';
        videoHeaderEl.style.display = 'none';
        videoListEl.innerHTML = '';

        try {
            const data = await go.main.App.SearchUser(keyword);

            if (data.type === 'uid') {
                this.fetchUserVideos(data.uid);
            } else if (data.type === 'users') {
                this.state.users = data.users;
                this.renderUserList();
                userResultsEl.style.display = 'block';
            }
        } catch (e) {
            this.toast('搜索失败: ' + e, 'error');
        }
    },

    renderUserList() {
        const container = document.getElementById('userList');
        container.innerHTML = this.state.users.map(u => `
            <div class="user-card" onclick="app.fetchUserVideos('${u.mid}')">
                <img class="user-avatar" src="${u.upic}" alt="${u.uname}">
                <h4>${u.uname}</h4>
                <div class="card-meta" style="justify-content:center; margin-top:5px;">
                    UID: ${u.mid} | 粉丝: ${u.fans}
                </div>
            </div>
        `).join('');
    },

    async fetchUserVideos(uid, page = 1) {
        this.state.currentUID = uid;
        this.state.currentPage = page;
        this.toast('正在获取视频列表...', 'info');

        try {
            const data = await go.main.App.GetUserVideos(uid, page);
            this.state.remoteVideos = data.videos || [];

            document.getElementById('videoListHeader').style.display = 'flex';
            document.getElementById('pageInfo').innerText = page;
            this.renderRemoteList();
        } catch (e) {
            this.toast('获取视频列表失败: ' + e, 'error');
        }
    },

    prevPage() {
        if (this.state.currentPage > 1 && this.state.currentUID) {
            this.fetchUserVideos(this.state.currentUID, this.state.currentPage - 1);
        }
    },

    nextPage() {
        if (this.state.currentUID) {
            this.fetchUserVideos(this.state.currentUID, this.state.currentPage + 1);
        }
    },

    triggerDownload(bvid, title) {
        if (this.state.downloadingParams.has(bvid)) return;

        go.main.App.DownloadVideo(bvid, title);
        this.state.downloadingParams.add(bvid);
        this.renderRemoteList();
        this.toast('已加入下载队列', 'success');
    },

    // --- Rendering ---
    renderLocalList() {
        const container = document.getElementById('localVideoList');
        if (this.state.localTasks.length === 0) {
            container.innerHTML = '<div class="empty-state">未找到视频 (请检查目录设置)</div>';
            return;
        }

        container.innerHTML = this.state.localTasks.map((task, idx) => {
            const isProcessing = task.Status.includes('处理中') || task.Status.includes('SUCCESS');
            const statusClass = task.Status.includes('SUCCESS') ? 'text-success' : '';

            let statusText = task.Status || '未处理';
            if (task.Status === 'SUCCESS' || task.Status.includes('SUCCESS')) statusText = '完成';
            else if (task.Status.includes('失败')) statusText = '失败';

            return `
            <div class="card">
                <div style="position:relative;">
                    <img class="card-img-top" src="${task.Info.coverPath}" onerror="this.src='data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxMDAiIGhlaWdodD0iMTAwIj48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSIjMjUyNjNhIi8+PC9zdmc+'">
                    <div style="position:absolute; bottom:8px; right:8px; background:rgba(0,0,0,0.7); color:#fff; padding:2px 6px; border-radius:4px; font-size:11px;">${task.Info.uname}</div>
                </div>
                <div class="card-body">
                    <h5 class="card-title" title="${task.Info.title}">${task.Info.title}</h5>
                    <div class="card-meta">
                        <span class="${statusClass}">${statusText}</span>
                    </div>
                    <div class="card-actions">
                        <button class="btn btn-standard btn-sm" onclick="app.triggerOpenFolder('${task.Dir.replace(/\\/g, '\\\\')}')" title="打开文件夹">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                            </svg>
                        </button>
                        <button class="btn btn-primary btn-sm" 
                            onclick="app.triggerMerge(${idx})"
                            ${isProcessing ? 'disabled' : ''}>
                            ${isProcessing ? '处理中' : '合并导出'}
                        </button>
                    </div>
                </div>
            </div>`;
        }).join('');
    },

    triggerOpenFolder(path) {
        go.main.App.OpenFolder(path).catch(e => {
            this.toast('无法打开文件夹: ' + e, 'error');
        });
    },

    renderRemoteList() {
        const container = document.getElementById('remoteVideoList');
        container.innerHTML = this.state.remoteVideos.map(v => {
            const isDownloading = this.state.downloadingParams.has(v.bvid);
            return `
            <div class="card">
                <div style="position:relative;">
                    <img class="card-img-top" src="${v.pic}" referrerpolicy="no-referrer">
                    <div style="position:absolute; bottom:8px; right:8px; background:rgba(0,0,0,0.7); color:#fff; padding:2px 6px; border-radius:4px; font-size:11px;">${v.length}</div>
                </div>
                <div class="card-body">
                    <h5 class="card-title" title="${v.title}">${v.title}</h5>
                    <div class="card-meta">
                        <span>${new Date(v.created * 1000).toLocaleDateString()}</span>
                    </div>
                    <div class="card-actions">
                        <button class="btn btn-primary btn-sm" 
                            onclick="app.triggerDownload('${v.bvid}', '${v.title.replace(/'/g, "\\'")}')"
                            ${isDownloading ? 'disabled' : ''}>
                            ${isDownloading ? '下载中...' : '下载'}
                        </button>
                    </div>
                </div>
            </div>`;
        }).join('');
    },

    // --- Utils ---
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

// Start
window.onload = () => {
    try {
        if (!app) throw new Error("App object not defined");
        app.init();
    } catch (e) {
        document.body.innerHTML = `<div style="color:red; padding:20px;">
            <h1>Startup Error</h1>
            <pre>${e.message}\n${e.stack}</pre>
        </div>`;
    }
};
