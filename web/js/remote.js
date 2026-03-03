// 远程下载视图：用户搜索 / 视频列表 / 下载 / 分P选择弹窗

Object.assign(app, {
    onRemoteKeywordInput() {
        if (this.state.remoteSearchTimer) {
            clearTimeout(this.state.remoteSearchTimer);
        }
        this.state.remoteSearchTimer = setTimeout(() => {
            const keyword = (document.getElementById('searchInput').value || '').trim();
            if (keyword.length >= 2) {
                this.searchRemote();
            }
        }, 350);
    },

    setRemoteLoading(loading, message = '正在搜索...') {
        this.state.remoteSearching = loading;
        const inputEl = document.getElementById('searchInput');
        const typeEl = document.getElementById('searchTypeSelect');
        const sortEl = document.getElementById('sortTypeSelect');
        if (inputEl) inputEl.disabled = false; // 保持可输入，避免“卡死感”
        if (typeEl) typeEl.disabled = loading;
        if (sortEl) sortEl.disabled = loading;
        if (loading) {
            document.getElementById('remoteVideoList').innerHTML = `<div class="empty-state">${message}</div>`;
        }
    },

    onRemoteSearchTypeChanged() {
        const mode = (document.getElementById('searchTypeSelect').value || 'user').trim();
        const input = document.getElementById('searchInput');
        if (input) {
            input.placeholder = mode === 'video'
                ? '输入视频关键词，或 BV/AV/CV...'
                : '输入 UID 或 UP主昵称...';
        }
    },

    onRemoteSortChanged() {
        if (this.state.remoteSearching) return;
        const order = (document.getElementById('sortTypeSelect').value || 'pubdate').trim();
        this.state.remoteSort = order;
        if (this.state.remoteSearchMode === 'video' && this.state.remoteKeyword) {
            this.fetchVideosByKeyword(this.state.remoteKeyword, 1, order);
            return;
        }
        if (this.state.currentUID) {
            this.fetchUserVideos(this.state.currentUID, 1, order);
        }
    },

    // --- 搜索 ---
    async searchRemote() {
        if (this.state.remoteSearching) return;
        const keyword = document.getElementById('searchInput').value.trim();
        if (!keyword) return;
        const mode = (document.getElementById('searchTypeSelect').value || 'user').trim();
        const order = (document.getElementById('sortTypeSelect').value || 'pubdate').trim();

        this.state.remoteSearchMode = mode;
        this.state.remoteKeyword = keyword;
        this.state.remoteSort = order;

        this.state.selectedRemote.clear();
        this.state.downloadQueue = [];
        this.state.batchDownloading = false;
        this.state.currentUID = '';

        document.getElementById('userResults').classList.add('hidden');
        document.getElementById('videoListHeader').classList.add('hidden');
        this.setRemoteLoading(true, '正在搜索...');

        try {
            if (mode === 'video') {
                await this.fetchVideosByKeyword(keyword, 1, order);
                return;
            }

            const data = await go.main.App.SearchUser(keyword);
            if (data.type === 'uid') {
                this.fetchUserVideos(data.uid, 1, order);
            } else if (data.type === 'users') {
                this.state.users = data.users || [];
                this.state.remoteVideos = [];
                this.state.remoteTotal = 0;
                this.state.remoteTotalPages = 0;
                this.renderRemoteList();
                if (this.state.users.length > 0) {
                    this.renderUserList();
                    document.getElementById('userResults').classList.remove('hidden');
                } else {
                    this.toast('未找到匹配的UP主', 'info');
                }
            }
        } catch (e) {
            this.toast('搜索失败: ' + e, 'error');
        } finally {
            this.setRemoteLoading(false);
        }
    },

    renderUserList() {
        const container = document.getElementById('userList');
        container.innerHTML = this.state.users.map(u => `
            <div class="user-card" onclick="app.fetchUserVideos('${u.mid}', 1, app.state.remoteSort || 'pubdate')">
                <img class="user-avatar" src="${u.upic}" referrerpolicy="no-referrer" alt="${u.uname}">
                <h4>${u.uname}</h4>
                <div class="card-meta user-meta-line">
                    UID: ${u.mid} | 粉丝: ${u.fans}
                </div>
            </div>
        `).join('');
    },

    async fetchUserVideos(uid, page = 1, order = 'pubdate') {
        this.state.currentUID = uid;
        this.state.currentPage = page;
        this.state.remoteSearchMode = 'user';
        this.state.remoteSort = order || 'pubdate';
        this.toast('正在获取视频列表...', 'info');
        this.setRemoteLoading(true, '正在获取视频列表...');

        try {
            const data = await go.main.App.GetUserVideos(uid, page, this.state.remotePageSize, this.state.remoteSort);
            this.state.remoteVideos = data.videos || [];
            this.state.remoteTotal = data.total || 0;
            this.state.remoteTotalPages = data.total_pages || 0;
            document.getElementById('videoListHeader').classList.remove('hidden');
            this.updateRemotePager();
            this.renderRemoteList();
        } catch (e) {
            this.toast('获取视频列表失败: ' + e, 'error');
        } finally {
            this.setRemoteLoading(false);
        }
    },

    async fetchVideosByKeyword(keyword, page = 1, order = 'pubdate') {
        this.state.currentPage = page;
        this.state.remoteSearchMode = 'video';
        this.state.remoteKeyword = keyword;
        this.state.remoteSort = order || 'pubdate';
        this.toast('正在搜索视频...', 'info');
        this.setRemoteLoading(true, '正在搜索视频...');
        try {
            const data = await go.main.App.SearchVideos(keyword, page, this.state.remotePageSize, this.state.remoteSort);
            this.state.users = [];
            this.state.remoteVideos = data.videos || [];
            this.state.remoteTotal = data.total || 0;
            this.state.remoteTotalPages = data.total_pages || 0;
            document.getElementById('videoListHeader').classList.remove('hidden');
            this.updateRemotePager();
            this.renderRemoteList();
            if (this.state.remoteVideos.length === 0) {
                this.toast('未找到匹配视频', 'info');
            }
        } catch (e) {
            this.toast('视频搜索失败: ' + e, 'error');
        } finally {
            this.setRemoteLoading(false);
        }
    },

    fetchCurrentPage(page) {
        if (this.state.remoteSearchMode === 'video') {
            if (!this.state.remoteKeyword) return;
            this.fetchVideosByKeyword(this.state.remoteKeyword, page, this.state.remoteSort);
            return;
        }
        if (!this.state.currentUID) return;
        this.fetchUserVideos(this.state.currentUID, page, this.state.remoteSort);
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
        if (this.state.currentPage > 1) {
            this.fetchCurrentPage(this.state.currentPage - 1);
        }
    },

    nextPage() {
        if (this.state.remoteTotalPages === 0 || this.state.currentPage < this.state.remoteTotalPages) {
            this.fetchCurrentPage(this.state.currentPage + 1);
        }
    },

    firstPage() {
        if (this.state.currentPage !== 1) {
            this.fetchCurrentPage(1);
        }
    },

    lastPage() {
        if (this.state.remoteTotalPages <= 0) return;
        if (this.state.currentPage !== this.state.remoteTotalPages) {
            this.fetchCurrentPage(this.state.remoteTotalPages);
        }
    },

    jumpToPage() {
        if (this.state.remoteSearchMode !== 'video' && !this.state.currentUID) return;
        const input = document.getElementById('pageJumpInput');
        const target = Number(input.value || 0);
        if (!Number.isInteger(target) || target <= 0) {
            this.toast('请输入有效页码', 'error');
            return;
        }
        const maxPage = this.state.remoteTotalPages || target;
        const page = Math.min(target, maxPage);
        this.fetchCurrentPage(page);
    },

    setPageSize() {
        const sel = document.getElementById('pageSizeSelect');
        const size = Number(sel.value || 30);
        if (!Number.isInteger(size) || size <= 0) return;
        this.state.remotePageSize = size;
        this.fetchCurrentPage(1);
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
                <div class="card-thumb-wrap">
                    <label class="remote-select">
                        <input type="checkbox" ${isSelected ? 'checked' : ''} onclick="app.toggleRemoteSelect(event, '${v.bvid}', '${v.title.replace(/'/g, "\\'").replace(/"/g, '&quot;')}', '${v.length || ''}')">
                        <span>选择</span>
                    </label>
                    <img class="card-img-top" src="${v.pic}" referrerpolicy="no-referrer">
                    <div class="card-badge">${v.length}</div>
                </div>
                <div class="card-body">
                    <h5 class="card-title" title="${v.title}">${v.title}</h5>
                    <div class="card-meta">
                        <span>${new Date(v.created * 1000).toLocaleDateString()}</span>
                        <span class="meta-stat">
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                                <path d="M3 7C3 5.89543 3.89543 5 5 5H15C16.1046 5 17 5.89543 17 7V17C17 18.1046 16.1046 19 15 19H5C3.89543 19 3 18.1046 3 17V7Z" stroke="currentColor" stroke-width="1.8"/>
                                <path d="M21 8L17 11V13L21 16V8Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
                            </svg>
                            ${v.play || '--'}
                        </span>
                        <span class="meta-stat">
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                                <path d="M5 6H19C20.1046 6 21 6.89543 21 8V14C21 15.1046 20.1046 16 19 16H11L7 19V16H5C3.89543 16 3 15.1046 3 14V8C3 6.89543 3.89543 6 5 6Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
                            </svg>
                            ${v.danmaku || '--'}
                        </span>
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
        document.getElementById('pageSelectModal').classList.remove('hidden');
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
        document.getElementById('pageSelectModal').classList.add('hidden');
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
