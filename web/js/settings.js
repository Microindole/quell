// 配置读写

Object.assign(app, {
    async loadConfig() {
        try {
            const cfg = await go.main.App.GetConfig();
            this.state.config = cfg;
            document.getElementById('cfgBiliDir').value   = cfg.bili_dir    || '';
            document.getElementById('cfgOutputDir').value = cfg.output_dir  || '';
            document.getElementById('cfgFFmpeg').value    = cfg.ffmpeg_path || '';
            document.getElementById('cfgSessdata').value  = cfg.sessdata    || '';
            const fmt = document.getElementById('cfgOutputFormat');
            fmt.value = cfg.output_format || 'mp4';
        } catch (e) {
            this.toast('无法加载配置: ' + e, 'error');
        }
    },

    async browseBiliDir() {
        const path = await go.main.App.OpenDirectoryDialog();
        if (path) document.getElementById('cfgBiliDir').value = path;
    },

    async browseOutputDir() {
        const path = await go.main.App.SelectOutputDir();
        if (path) document.getElementById('cfgOutputDir').value = path;
    },

    async browseFFmpeg() {
        const path = await go.main.App.OpenFileDialog();
        if (path) document.getElementById('cfgFFmpeg').value = path;
    },

    async saveConfig() {
        const payload = {
            bili_dir:      document.getElementById('cfgBiliDir').value,
            output_dir:    document.getElementById('cfgOutputDir').value,
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
    },

    async startBiliLogin() {
        const modal = document.getElementById('biliLoginModal');
        const statusEl = document.getElementById('biliLoginStatus');
        const urlEl = document.getElementById('biliLoginURL');
        const qrEl = document.getElementById('biliLoginQrImage');
        statusEl.innerText = '正在初始化登录会话...';
        urlEl.value = '';
        qrEl.removeAttribute('src');
        modal.style.display = 'flex';

        try {
            const data = await go.main.App.StartBiliLogin();
            this.state.loginQrcodeKey = data.qrcode_key;
            this.state.loginURL = data.url;
            urlEl.value = data.url || '';
            qrEl.src = `https://api.qrserver.com/v1/create-qr-code/?size=260x260&data=${encodeURIComponent(data.url || '')}`;
            statusEl.innerText = '请用手机 B站 App 扫码并确认登录。';
            this.startLoginPolling();
        } catch (e) {
            statusEl.innerText = '初始化失败: ' + e;
            this.toast('登录初始化失败: ' + e, 'error');
        }
    },

    openLoginBrowser() {
        if (this.state.loginURL) {
            go.main.App.OpenBrowserURL(this.state.loginURL);
        }
    },

    closeBiliLoginModal() {
        const modal = document.getElementById('biliLoginModal');
        if (modal) modal.style.display = 'none';
        const qrEl = document.getElementById('biliLoginQrImage');
        if (qrEl) qrEl.removeAttribute('src');
        if (this.state.loginPollingTimer) {
            clearInterval(this.state.loginPollingTimer);
            this.state.loginPollingTimer = null;
        }
        this.state.loginQrcodeKey = '';
        this.state.loginURL = '';
    },

    startLoginPolling() {
        if (this.state.loginPollingTimer) {
            clearInterval(this.state.loginPollingTimer);
        }

        this.state.loginPollingTimer = setInterval(async () => {
            const key = this.state.loginQrcodeKey;
            if (!key) return;

            try {
                const res = await go.main.App.PollBiliLogin(key);
                const statusEl = document.getElementById('biliLoginStatus');
                if (statusEl) statusEl.innerText = res.message || '登录处理中...';

                if (res.status === 'success') {
                    clearInterval(this.state.loginPollingTimer);
                    this.state.loginPollingTimer = null;
                    await this.loadConfig();
                    this.toast('B站登录成功，SESSDATA 已自动保存', 'success');
                    this.closeBiliLoginModal();
                } else if (res.status === 'expired') {
                    clearInterval(this.state.loginPollingTimer);
                    this.state.loginPollingTimer = null;
                    this.toast('二维码已过期，请重新发起登录', 'error');
                }
            } catch (e) {
                clearInterval(this.state.loginPollingTimer);
                this.state.loginPollingTimer = null;
                const statusEl = document.getElementById('biliLoginStatus');
                if (statusEl) statusEl.innerText = '登录轮询失败: ' + e;
                this.toast('登录轮询失败: ' + e, 'error');
            }
        }, 1200);
    }
});
