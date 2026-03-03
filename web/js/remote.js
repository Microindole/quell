// 远程下载视图：用户搜索 / 视频列表 / 下载 / 分P选择弹窗

Object.assign(app, {
    // --- 搜索 ---
    async searchRemote() {
        const keyword = document.getElementById('searchInput').value.trim();
        if (!keyword) return;
        this.state.selectedRemote.clear();
        this.state.downloadQueue = [];
        this.state.batchDownloading = false;

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
            const data = await go.main.App.GetUserVideos(uid, page, this.state.remotePageSize);
            this.state.remoteVideos = data.videos || [];
            this.state.remoteTotal = data.total || 0;
            this.state.remoteTotalPages = data.total_pages || 0;
            document.getElementById('videoListHeader').style.display = 'flex';
            this.updateRemotePager();
            this.renderRemoteList();
        } catch (e) {
            this.toast('获取视频列表失败: ' + e, 'error');
        }
    },

    updateRemotePager() {
        const page = this.state.currentPage;
        const totalPages = this.state.remoteTotalPages || 0;
        const total = this.state.remoteTotal || 0;
        document.getElementById('pageInfo').innerText = `第 ${page} / ${Math.max(totalPages, 1)} 页，共 ${total} 个`;

        const prevDisabled = page <= 1;
        const nextDisabled = totalPages > 0 ? page >= totalPages : false;
        document.getElementById('btnFirstPage').disabled = prevDisabled;
        document.getElementById('btnPrevPage').disabled = prevDisabled;
        document.getElementById('btnNextPage').disabled = nextDisabled;
        document.getElementById('btnLastPage').disabled = nextDisabled || totalPages === 0;

        const jumpInput = document.getElementById('pageJumpInput');
        if (jumpInput) jumpInput.value = page;
        const selectedCountEl = document.getElementById('selectedCount');
        if (selectedCountEl) selectedCountEl.innerText = `${this.state.selectedRemote.size}`;
    },

    prevPage() {
        if (this.state.currentPage > 1 && this.state.currentUID) {
            this.fetchUserVideos(this.state.currentUID, this.state.currentPage - 1);
        }
    },

    nextPage() {
        if (this.state.currentUID && (this.state.remoteTotalPages === 0 || this.state.currentPage < this.state.remoteTotalPages)) {
            this.fetchUserVideos(this.state.currentUID, this.state.currentPage + 1);
        }
    },

    firstPage() {
        if (this.state.currentUID && this.state.currentPage !== 1) {
            this.fetchUserVideos(this.state.currentUID, 1);
        }
    },

    lastPage() {
        if (!this.state.currentUID || this.state.remoteTotalPages <= 0) return;
        if (this.state.currentPage !== this.state.remoteTotalPages) {
            this.fetchUserVideos(this.state.currentUID, this.state.remoteTotalPages);
        }
    },

    jumpToPage() {
        if (!this.state.currentUID) return;
        const input = document.getElementById('pageJumpInput');
        const target = Number(input.value || 0);
        if (!Number.isInteger(target) || target <= 0) {
            this.toast('请输入有效页码', 'error');
            return;
        }
        const maxPage = this.state.remoteTotalPages || target;
        const page = Math.min(target, maxPage);
        this.fetchUserVideos(this.state.currentUID, page);
    },

    setPageSize() {
        const sel = document.getElementById('pageSizeSelect');
        const size = Number(sel.value || 30);
        if (!Number.isInteger(size) || size <= 0) return;
        this.state.remotePageSize = size;
        if (this.state.currentUID) {
            this.fetchUserVideos(this.state.currentUID, 1);
        }
    },

    renderRemoteList() {
        const container = document.getElementById('remoteVideoList');
        if (this.state.remoteVideos.length === 0) {
            container.innerHTML = '<div class="empty-state">当前页没有视频</div>';
            this.updateRemotePager();
            return;
        }

        container.innerHTML = this.state.remoteVideos.map(v => {
            const isDownloading = this.state.downloadingParams.has(v.bvid);
            const isSelected = this.state.selectedRemote.has(v.bvid);
            return `
            <div class="card">
                <div style="position:relative;">
                    <label class="remote-select">
                        <input type="checkbox" ${isSelected ? 'checked' : ''} onclick="app.toggleRemoteSelect(event, '${v.bvid}', '${v.title.replace(/'/g, "\\'").replace(/"/g, '&quot;')}', '${v.length || ''}')">
                        <span>选择</span>
                    </label>
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
                            onclick="app.triggerDownload('${v.bvid}', '${v.title.replace(/'/g, "\\'").replace(/"/g, '&quot;')}', '${v.length || ''}')"
                            ${isDownloading ? 'disabled' : ''}>
                            ${isDownloading ? '下载中...' : '下载'}
                        </button>
                    </div>
                </div>
            </div>`;
        }).join('');
        this.updateRemotePager();
    },

    toggleRemoteSelect(event, bvid, title, length) {
        event.stopPropagation();
        if (this.state.selectedRemote.has(bvid)) {
            this.state.selectedRemote.delete(bvid);
        } else {
            this.state.selectedRemote.set(bvid, { title, length: length || '' });
        }
        this.updateRemotePager();
    },

    selectCurrentPage() {
        this.state.remoteVideos.forEach(v => this.state.selectedRemote.set(v.bvid, { title: v.title, length: v.length || '' }));
        this.renderRemoteList();
    },

    clearSelectedRemote() {
        this.state.selectedRemote.clear();
        this.renderRemoteList();
    },

    startBatchDownload() {
        if (this.state.batchDownloading) {
            this.toast('批量下载正在进行中', 'info');
            return;
        }
        const queue = [...this.state.selectedRemote.entries()]
            .filter(([bvid]) => !this.state.downloadingParams.has(bvid))
            .map(([bvid, meta]) => ({ bvid, title: meta.title, length: meta.length || '' }));

        if (queue.length === 0) {
            this.toast('请先选择要下载的视频', 'error');
            return;
        }

        this.state.downloadQueue = queue;
        this.state.batchDownloading = true;
        this.toast(`已加入队列 ${queue.length} 个，开始逐个下载`, 'success');
        this.runNextInQueue();
    },

    runNextInQueue() {
        if (!this.state.batchDownloading) return;
        if (this.state.downloadQueue.length === 0) {
            this.state.batchDownloading = false;
            this.toast('批量下载队列已完成', 'success');
            return;
        }

        const next = this.state.downloadQueue.shift();
        this.state.downloadingParams.add(next.bvid);
        this.renderRemoteList();
        go.main.App.DownloadVideo(next.bvid, next.title, next.length || '', { quality_id: 0, codec: '' });
    },

    onRemoteDownloadFinished(bvid) {
        if (this.state.selectedRemote.has(bvid)) {
            this.state.selectedRemote.delete(bvid);
        }
        this.updateRemotePager();
        if (this.state.batchDownloading) {
            this.runNextInQueue();
        }
    },

    // --- 下载入口（先获取分P信息） ---
    async triggerDownload(bvid, title, length) {
        if (this.state.downloadingParams.has(bvid)) return;

        try {
            const info = await go.main.App.GetVideoPages(bvid);
            if (!info || !info.pages || info.pages.length === 0) {
                this.toast('未获取到可下载分P', 'error');
                return;
            }
            this.showPageModal(bvid, title, length || '', info);
        } catch (e) {
            this.toast('获取视频信息失败: ' + e, 'error');
        }
    },

    // --- 分P选择弹窗 ---
    showPageModal(bvid, title, length, info) {
        this.state.pageSelectBvid = bvid;
        this.state.pageSelectTitle = title;
        this.state.pageSelectLength = length || '';
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
            go.main.App.DownloadVideo(bvid, title, this.state.pageSelectLength || '', pref);
        } else {
            go.main.App.DownloadVideoPages(bvid, selected, title, pref);
        }

        this.state.downloadingParams.add(bvid);
        this.renderRemoteList();
        this.toast(`开始下载 ${selected.length} 个分P`, 'success');
        this.closePageModal();
    }
});
