// 远程下载视图：用户搜索 / 视频列表 / 下载 / 分P选择弹窗

Object.assign(app, {
    // --- 搜索 ---
    async searchRemote() {
        const keyword = document.getElementById('searchInput').value.trim();
        if (!keyword) return;

        document.getElementById('userResults').style.display = 'none';
        document.getElementById('videoListHeader').style.display = 'none';
        document.getElementById('remoteVideoList').innerHTML = '';

        try {
            const data = await go.main.App.SearchUser(keyword);
            if (data.type === 'uid') {
                this.fetchUserVideos(data.uid);
            } else if (data.type === 'users') {
                this.state.users = data.users;
                this.renderUserList();
                document.getElementById('userResults').style.display = 'block';
            }
        } catch (e) {
            this.toast('搜索失败: ' + e, 'error');
        }
    },

    renderUserList() {
        const container = document.getElementById('userList');
        container.innerHTML = this.state.users.map(u => `
            <div class="user-card" onclick="app.fetchUserVideos('${u.mid}')">
                <img class="user-avatar" src="${u.upic}" referrerpolicy="no-referrer" alt="${u.uname}">
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
                            onclick="app.triggerDownload('${v.bvid}', '${v.title.replace(/'/g, "\\'").replace(/"/g, '&quot;')}')"
                            ${isDownloading ? 'disabled' : ''}>
                            ${isDownloading ? '下载中...' : '下载'}
                        </button>
                    </div>
                </div>
            </div>`;
        }).join('');
    },

    // --- 下载入口（先获取分P信息） ---
    async triggerDownload(bvid, title) {
        if (this.state.downloadingParams.has(bvid)) return;

        try {
            const info = await go.main.App.GetVideoPages(bvid);
            if (!info || !info.pages || info.pages.length === 0) {
                this.toast('未获取到可下载分P', 'error');
                return;
            }
            this.showPageModal(bvid, title, info);
        } catch (e) {
            this.toast('获取视频信息失败: ' + e, 'error');
        }
    },

    // --- 分P选择弹窗 ---
    showPageModal(bvid, title, info) {
        this.state.pageSelectBvid = bvid;
        this.state.pageSelectTitle = title;
        this.state.pageSelectPages = info.pages;
        this.state.pageStreamOptions = info.stream_options || [];

        document.getElementById('pageSelectTitle').innerText = `选择分P - ${info.title}`;

        const listEl = document.getElementById('pageSelectList');
        listEl.innerHTML = info.pages.map(p => `
            <div class="page-item">
                <input type="checkbox" id="page_${p.page}" checked>
                <span class="page-num">P${p.page}</span>
                <label for="page_${p.page}">${p.part || ('第' + p.page + '部分')}</label>
            </div>
        `).join('');

        this.renderStreamOptions();
        document.getElementById('pageSelectModal').style.display = 'flex';
    },

    renderStreamOptions() {
        const qualityEl = document.getElementById('downloadQuality');
        const options = this.state.pageStreamOptions || [];

        const qualityMap = new Map();
        options.forEach(o => {
            if (!qualityMap.has(o.quality_id)) {
                qualityMap.set(o.quality_id, o.quality_label);
            }
        });

        const qualities = [...qualityMap.entries()];
        if (qualities.length === 0) {
            qualityEl.innerHTML = `<option value="0">自动最高可用</option>`;
            this.state.downloadPref.quality_id = 0;
            this.renderCodecOptions();
            return;
        }

        qualityEl.innerHTML = qualities.map(([qid, label]) => `<option value="${qid}">${label}</option>`).join('');

        if (qualities.length > 0) {
            this.state.downloadPref.quality_id = Number(qualities[0][0]);
        } else {
            this.state.downloadPref.quality_id = 0;
        }
        this.renderCodecOptions();
    },

    renderCodecOptions() {
        const codecEl = document.getElementById('downloadCodec');
        const qid = Number(document.getElementById('downloadQuality').value || 0);
        this.state.downloadPref.quality_id = qid;

        const codecMap = new Map();
        (this.state.pageStreamOptions || [])
            .filter(o => o.quality_id === qid)
            .forEach(o => {
                if (!codecMap.has(o.codec)) {
                    codecMap.set(o.codec, o.codec_label);
                }
            });

        const codecs = [...codecMap.entries()];
        if (codecs.length === 0) {
            codecEl.innerHTML = `<option value="">自动选择</option>`;
            this.state.downloadPref.codec = '';
            return;
        }

        codecEl.innerHTML = codecs.map(([codec, label]) => `<option value="${codec}">${label}</option>`).join('');
        this.state.downloadPref.codec = codecs[0][0];
    },

    onQualityChanged() {
        this.renderCodecOptions();
    },

    onCodecChanged() {
        this.state.downloadPref.codec = document.getElementById('downloadCodec').value || '';
    },

    closePageModal() {
        document.getElementById('pageSelectModal').style.display = 'none';
    },

    selectAllPages() {
        document.querySelectorAll('#pageSelectList input[type="checkbox"]').forEach(cb => {
            cb.checked = true;
        });
    },

    downloadSelectedPages() {
        const pages = this.state.pageSelectPages;
        const selected = pages.filter(p => {
            const cb = document.getElementById(`page_${p.page}`);
            return cb && cb.checked;
        });

        if (selected.length === 0) {
            this.toast('请至少选择一个分P', 'error');
            return;
        }

        const bvid = this.state.pageSelectBvid;
        const title = this.state.pageSelectTitle;
        const pref = {
            quality_id: Number(document.getElementById('downloadQuality').value || 0),
            codec: document.getElementById('downloadCodec').value || ''
        };

        if (selected.length === pages.length) {
            go.main.App.DownloadVideo(bvid, title, pref);
        } else {
            go.main.App.DownloadVideoPages(bvid, selected, title, pref);
        }

        this.state.downloadingParams.add(bvid);
        this.renderRemoteList();
        this.toast(`开始下载 ${selected.length} 个分P`, 'success');
        this.closePageModal();
    }
});
