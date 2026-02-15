// 配置读写

Object.assign(app, {
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
            BiliDir:    document.getElementById('cfgBiliDir').value,
            FFmpegPath: document.getElementById('cfgFFmpeg').value,
            SESSDATA:   document.getElementById('cfgSessdata').value
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
