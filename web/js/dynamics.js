// 动态流视图：拉取UP动态 / 复制文字 / 下载图片 / 下载动态视频

Object.assign(app, {
    dynamicTypeLabel(rawType) {
        const t = String(rawType || '').trim().toUpperCase();
        const map = {
            DYNAMIC_TYPE_AV: '视频',
            DYNAMIC_TYPE_DRAW: '图文',
            DYNAMIC_TYPE_WORD: '文字',
            DYNAMIC_TYPE_ARTICLE: '专栏',
            DYNAMIC_TYPE_FORWARD: '转发',
            DYNAMIC_TYPE_LIVE_RCMD: '直播',
            DYNAMIC_TYPE_COMMON_SQUARE: '动态',
            DYNAMIC_TYPE_NONE: '动态'
        };
        return map[t] || '动态';
    },

    escapeHTML(text) {
        return String(text || '')
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    },

    safeURL(raw) {
        const s = String(raw || '').trim();
        if (!s) return '';
        if (s.startsWith('https://') || s.startsWith('http://')) return s;
        if (s.startsWith('//')) return 'https:' + s;
        return '';
    },

    renderDynamicRichText(item) {
        const nodes = Array.isArray(item.rich_nodes) ? item.rich_nodes : [];
        if (nodes.length === 0) return this.escapeHTML(item.text || '');
        return nodes.map((n) => {
            const type = String(n.type || '').toUpperCase();
            const text = this.escapeHTML(n.text || '');
            const jump = this.safeURL(n.jump_url || '');
            const emojiURL = this.safeURL(n.emoji_url || '');

            if (type.includes('EMOJI') && emojiURL) {
                const alt = text || '[表情]';
                return `<img class="dynamic-emoji" src="${emojiURL}" alt="${alt}" title="${alt}" referrerpolicy="no-referrer">`;
            }
            if ((type.includes('AT') || type.includes('TOPIC') || type.includes('WEB') || type.includes('LOTTERY') || type.includes('BV')) && jump) {
                return `<a class="dynamic-rich-link" href="${jump}" target="_blank" rel="noreferrer noopener">${text || jump}</a>`;
            }
            return text;
        }).join('');
    },

    clearDynamics() {
        this.state.dynamicOffset = '';
        this.state.dynamicHasMore = false;
        this.state.dynamics = [];
        const listEl = document.getElementById('dynamicList');
        if (listEl) listEl.innerHTML = '<div class="empty-state">暂无动态</div>';
        const moreBtn = document.getElementById('btnMoreDynamics');
        if (moreBtn) moreBtn.classList.add('hidden');
    },

    async fetchDynamics() {
        if (this.state.dynamicLoading) return;
        if (Date.now() < (this.state.dynamicCooldownUntil || 0)) {
            const sec = Math.ceil((this.state.dynamicCooldownUntil - Date.now()) / 1000);
            this.toast(`请求过快，请 ${sec}s 后重试`, 'info');
            return;
        }
        const uid = (document.getElementById('dynamicUIDInput').value || '').trim();
        if (!uid) {
            this.toast('请输入UP主UID', 'error');
            return;
        }
        this.state.dynamicUID = uid;
        this.state.dynamicOffset = '';
        this.state.dynamicHasMore = false;
        this.state.dynamics = [];
        await this.loadDynamicsPage(true);
    },

    async loadMoreDynamics() {
        if (this.state.dynamicLoading) return;
        if (Date.now() < (this.state.dynamicCooldownUntil || 0)) {
            const sec = Math.ceil((this.state.dynamicCooldownUntil - Date.now()) / 1000);
            this.toast(`请求过快，请 ${sec}s 后重试`, 'info');
            return;
        }
        if (!this.state.dynamicUID) {
            this.toast('请先输入UID并获取动态', 'error');
            return;
        }
        if (!this.state.dynamicHasMore) {
            this.toast('没有更多动态了', 'info');
            return;
        }
        await this.loadDynamicsPage(false);
    },

    async loadDynamicsPage(reset) {
        if (this.state.dynamicLoading) return;
        this.state.dynamicLoading = true;
        const listEl = document.getElementById('dynamicList');
        const moreBtn = document.getElementById('btnMoreDynamics');
        const loadBtn = document.getElementById('btnMoreDynamics');
        if (loadBtn) loadBtn.disabled = true;
        if (listEl && (reset || this.state.dynamics.length === 0)) {
            listEl.innerHTML = '<div class="empty-state">正在加载动态...</div>';
        }
        try {
            const data = await go.main.App.GetUserDynamics(this.state.dynamicUID, this.state.dynamicOffset || '');
            const items = data.items || [];
            if (reset) {
                this.state.dynamics = items;
            } else {
                this.state.dynamics = this.state.dynamics.concat(items);
            }
            this.state.dynamics.sort((a, b) => Number(b.pub_ts || 0) - Number(a.pub_ts || 0));
            this.state.dynamicOffset = data.offset || '';
            this.state.dynamicHasMore = !!data.has_more;
            this.renderDynamics();
            if (moreBtn) {
                if (this.state.dynamicHasMore) moreBtn.classList.remove('hidden');
                else moreBtn.classList.add('hidden');
            }
        } catch (e) {
            const msg = String(e || '');
            if (msg.includes('code=-412')) {
                this.state.dynamicCooldownUntil = Date.now() + 8000;
                this.toast('触发风控，已进入8秒冷却，请稍后重试', 'error');
            } else {
                this.toast('获取动态失败: ' + e, 'error');
            }
            if (listEl && this.state.dynamics.length === 0) {
                listEl.innerHTML = '<div class="empty-state">动态加载失败</div>';
            }
        } finally {
            this.state.dynamicLoading = false;
            if (loadBtn) loadBtn.disabled = false;
        }
    },

    renderDynamics() {
        const listEl = document.getElementById('dynamicList');
        if (!listEl) return;
        if (!this.state.dynamics || this.state.dynamics.length === 0) {
            listEl.innerHTML = '<div class="empty-state">暂无动态</div>';
            return;
        }
        listEl.innerHTML = this.state.dynamics.map((d) => {
            const textHTML = this.renderDynamicRichText(d);
            const uname = this.escapeHTML(d.user_name || '未知UP');
            const pub = this.escapeHTML(d.pub_time || '');
            const typeText = this.dynamicTypeLabel(d.type || '');
            const imgCount = (d.image_urls || []).length;
            const bvid = this.escapeHTML(d.bvid || '');
            const title = this.escapeHTML(d.video_title || '');
            const avatar = this.escapeHTML(d.user_face || '');
            const images = (d.image_urls || []).slice(0, 9).map((u) => `<img src="${this.escapeHTML(u)}" referrerpolicy="no-referrer" alt="">`).join('');
            return `
            <article class="dynamic-item">
                <div class="dynamic-head">
                    <img class="dynamic-avatar" src="${avatar}" referrerpolicy="no-referrer" alt="${uname}">
                    <div class="dynamic-head-meta">
                        <div class="dynamic-name">${uname}</div>
                        <div class="dynamic-time">${pub || '--'} · ${typeText}</div>
                    </div>
                </div>
                <div class="dynamic-body">
                    <div class="dynamic-text">${textHTML}</div>
                    ${imgCount > 0 ? `<div class="dynamic-media-grid">${images}</div>` : ''}
                    ${bvid ? `<div class="dynamic-video-box">视频: ${title || bvid}</div>` : ''}
                    <div class="dynamic-actions">
                        <button class="dynamic-action-btn" onclick="app.copyDynamicText('${this.escapeHTML(d.id_str)}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true"><rect x="9" y="9" width="10" height="10" rx="2" stroke="currentColor" stroke-width="1.8"/><rect x="5" y="5" width="10" height="10" rx="2" stroke="currentColor" stroke-width="1.8"/></svg>
                            复制
                        </button>
                        ${imgCount > 0 ? `<button class="dynamic-action-btn" onclick="app.downloadDynamicImages('${this.escapeHTML(d.id_str)}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M12 4V14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/><path d="M8 10L12 14L16 10" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/><path d="M5 19H19" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>
                            图片(${imgCount})
                        </button>` : ''}
                        ${bvid ? `<button class="dynamic-action-btn dynamic-action-primary" onclick="app.downloadDynamicVideo('${bvid}', '${title}')">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 6C5 4.89543 5.89543 4 7 4H13C14.1046 4 15 4.89543 15 6V18C15 19.1046 14.1046 20 13 20H7C5.89543 20 5 19.1046 5 18V6Z" stroke="currentColor" stroke-width="1.8"/><path d="M19 8L15 11V13L19 16V8Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/></svg>
                            视频
                        </button>` : ''}
                    </div>
                </div>
            </article>`;
        }).join('');
    },

    findDynamicByID(id) {
        return (this.state.dynamics || []).find((d) => d.id_str === id);
    },

    async copyDynamicText(id) {
        const d = this.findDynamicByID(id);
        if (!d) return;
        const text = d.text || '';
        if (!text) {
            this.toast('该动态没有可复制文字', 'info');
            return;
        }
        try {
            await navigator.clipboard.writeText(text);
            this.toast('文字已复制', 'success');
        } catch (_) {
            const ta = document.createElement('textarea');
            ta.value = text;
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            ta.remove();
            this.toast('文字已复制', 'success');
        }
    },

    async downloadDynamicImages(id) {
        const d = this.findDynamicByID(id);
        if (!d) return;
        const urls = d.image_urls || [];
        if (urls.length === 0) {
            this.toast('该动态没有图片', 'info');
            return;
        }
        try {
            const saved = await go.main.App.DownloadDynamicMedia(id, urls);
            this.toast(`图片下载完成 (${saved.length})`, 'success');
            this.log(`[SUCCESS] 动态图片下载完成: ${id}`);
        } catch (e) {
            this.toast('图片下载失败: ' + e, 'error');
        }
    },

    downloadDynamicVideo(bvid, title) {
        if (!bvid) return;
        if (this.state.downloadingParams.has(bvid)) {
            this.toast('该视频已在下载中', 'info');
            return;
        }
        this.state.downloadingParams.add(bvid);
        go.main.App.DownloadVideo(bvid, title || bvid, '', { quality_id: 0, codec: '' });
        this.toast(`开始下载动态视频: ${bvid}`, 'success');
    }
});
