// 本地视频视图：扫描 / 合并 / 渲染

Object.assign(app, {
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

    triggerOpenFolder(path) {
        go.main.App.OpenFolder(path).catch(e => {
            this.toast('无法打开文件夹: ' + e, 'error');
        });
    },

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
                    <img class="card-img-top" src="${task.Info.coverPath}"
                        onerror="this.src='data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxMDAiIGhlaWdodD0iMTAwIj48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSIjMjUyNjNhIi8+PC9zdmc+'">
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
    }
});
