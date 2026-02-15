// Wails 后端事件监听（merge / download / progress）

Object.assign(app, {
    setupEvents() {
        const statusEl = document.getElementById('statusBar');

        window.runtime.EventsOn('merge', (data) => {
            if (data.status === 'processing') {
                this.toast(`正在合并: ${data.title}`, 'success');
            } else if (data.status === 'done') {
                const outPath = data.output || '合并完成';
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

        window.runtime.EventsOn('download', (data) => {
            if (data.status === 'done') {
                this.toast(`下载完成: ${data.bvid}`, 'success');
                this.state.downloadingParams.delete(data.bvid);
                this.renderRemoteList();
            } else if (data.status === 'error') {
                this.toast(`下载失败 [${data.bvid}]: ${data.error}`, 'error');
                this.state.downloadingParams.delete(data.bvid);
                this.renderRemoteList();
            }
        });

        window.runtime.EventsOn('progress', (data) => {
            if (statusEl) statusEl.innerText = `[${data.bvid}] ${data.message}`;
        });
    }
});
