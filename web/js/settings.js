// 配置读写

Object.assign(app, {
    async loadConfig() {
        try {
            const cfg = await go.main.App.GetConfig();
            this.state.config = cfg;
            document.getElementById('cfgBiliDir').value   = cfg.bili_dir    || '';
            document.getElementById('cfgFFmpeg').value    = cfg.ffmpeg_path || '';
            document.getElementById('cfgSessdata').value  = cfg.sessdata    || '';
            const fmt = document.getElementById('cfgOutputFormat');
            fmt.value = cfg.output_format || 'mp4';
        } catch (e) {
            this.toast('无法加载配置: ' + e, 'error');
        }
    },

    async saveConfig() {
        const payload = {
            bili_dir:      document.getElementById('cfgBiliDir').value,
            ffmpeg_path:   document.getElementById('cfgFFmpeg').value,
            sessdata:      document.getElementById('cfgSessdata').value,
            output_format: document.getElementById('cfgOutputFormat').value,
        };
        try {
            await go.main.App.SaveConfig(payload);
            this.state.config = payload;
            this.toast('配置已保存', 'success');
        } catch (e) {
            this.toast('保存配置失败: ' + e, 'error');
        }
    }
});
