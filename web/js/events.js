// Wails 后端事件监听（merge / download / progress）

Object.assign(app, {
    setupEvents() {
        const statusEl = document.getElementById('statusBar');

        window.runtime.EventsOn('merge', (data) => {
            if (data.status === 'processing') {
                this.toast(`正在合并: ${data.title}`, 'success');
            } else if (data.status === 'done') {
                const outPath = data.output || '合并完成';
                this.toast(`合并完成`, 'success');
                this.log(`[SUCCESS] 合并完成，文件位于: ${outPath}`);
                if (data.index >= 0 && data.index < this.state.localTasks.length) {
                    this.state.localTasks[data.index].Status = '完成';
                    this.state.localTasks[data.index].OutputPath = outPath;
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

        window.runtime.EventsOn('merge_progress', (data) => {
            if (statusEl) statusEl.innerText = `[合并中] ${data.message}`;
        });

        window.runtime.EventsOn('batch_merge', (data) => {
            if (data.status === 'started') {
                this.toast('批量合并已开始', 'success');
            } else if (data.status === 'done') {
                this.toast('批量合并完成', 'success');
                this.scanLocal(); // 刷新列表
            }
        });

        window.runtime.EventsOn('batch_merge_progress', (data) => {
            if (statusEl) statusEl.innerText = `[批量合并] ${data.message}`;
        });

        window.runtime.EventsOn('download', (data) => {
            if (data.status === 'started') {
                const dir = data.output_dir || '(未配置)';
                this.log(`[INFO] 开始下载: ${data.bvid}，保存目录: ${dir}`);
            } else if (data.status === 'done') {
                const dir = data.output_dir || '(未配置)';
                const tail = data.last_message ? `，最后状态: ${data.last_message}` : '';
                this.toast(`下载完成: ${data.bvid}`, 'success');
                this.log(`[SUCCESS] 下载完成: ${data.bvid}，保存目录: ${dir}${tail}`);
                this.state.downloadingParams.delete(data.bvid);
                this.renderRemoteList();
            } else if (data.status === 'error') {
                const dir = data.output_dir || '(未配置)';
                const tail = data.last_message ? `，最后状态: ${data.last_message}` : '';
                this.toast(`下载失败 [${data.bvid}]: ${data.error}`, 'error');
                this.log(`[ERROR] 下载失败: ${data.bvid}，目录: ${dir}，错误: ${data.error}${tail}`);
                this.state.downloadingParams.delete(data.bvid);
                this.renderRemoteList();
            }
        });

        window.runtime.EventsOn('progress', (data) => {
            if (statusEl) statusEl.innerText = `[${data.bvid}] ${data.message}`;
        });
    }
});
