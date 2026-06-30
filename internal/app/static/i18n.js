(() => {
  const storageKey = 'mcmon-lang';
  const zh = {
    'Dashboard': '仪表盘',
    'Servers': '服务器',
    'Remote': '远程',
    'Settings': '设置',
    'Toggle theme': '切换主题',
    'Online': '在线',
    'Offline': '离线',
    'Players': '人数',
    'Avg Latency': '平均延迟',
    'Latency': '延迟',
    'Loss': '丢包',
    'No servers configured yet.': '还没有配置服务器。',
    'Add one →': '添加一个 →',
    'Status': '状态',
    'Name': '名称',
    'Host': '地址',
    'Enabled monitors': '已启用监控',
    'Timeout': '超时',
    'Export selected': '导出选中项',
    'Import': '导入',
    'Select servers to export their configuration.': '选择要导出配置的服务器。',
    'Add server': '添加服务器',
    'Edit server': '编辑服务器',
    'Save changes': '保存更改',
    'Cancel': '取消',
    'Timeout (ms)': '超时（毫秒）',
    'Interval (sec)': '周期（秒）',
    'Probes / burst': '每轮探测次数',
    'Probe gap (ms)': '探测间隔（毫秒）',
    'Protocol ver.': '协议版本',
    'Import servers': '导入服务器',
    'Import selected': '导入选中项',
    'Server Detail': '服务器详情',
    'Back to Dashboard': '返回仪表盘',
    'Server not found': '未找到服务器',
    'App Settings': '应用设置',
    'Run in background': '后台运行',
    'Remote Host Connection': '远程 Host 连接',
    'Connect to a MCMon Host to view latency data collected by remote agents.': '连接到 MCMon Host 以查看远程节点采集的数据。',
    'Remote Servers': '远程服务器',
    'Remote Server Detail': '远程服务器详情',
    'Back to Remote': '返回远程',
    'Agents': '节点',
    'Authorized': '已授权',
    'Saved': '已保存',
    'Token required': '需要 token',
    'Remote error': '远程错误',
    'None': '无',
  };

  function defaultLanguage() {
    const saved = localStorage.getItem(storageKey);
    if (saved === 'en' || saved === 'zh-CN') return saved;
    return navigator.language && navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';
  }

  function dictionary() {
    return currentLanguage() === 'zh-CN' ? zh : {};
  }

  function currentLanguage() {
    return localStorage.getItem(storageKey) || defaultLanguage();
  }

  function translate(text) {
    const trimmed = String(text).trim();
    return dictionary()[trimmed] || text;
  }

  function translateTextNodes(root) {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
      acceptNode(node) {
        if (!node.nodeValue.trim()) return NodeFilter.FILTER_REJECT;
        const parent = node.parentElement;
        if (!parent || ['SCRIPT', 'STYLE', 'TEXTAREA', 'CODE', 'PRE'].includes(parent.tagName)) {
          return NodeFilter.FILTER_REJECT;
        }
        return NodeFilter.FILTER_ACCEPT;
      }
    });
    const nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    nodes.forEach(node => {
      const original = node.__i18nOriginal || node.nodeValue;
      node.__i18nOriginal = original;
      const leading = original.match(/^\s*/)[0];
      const trailing = original.match(/\s*$/)[0];
      node.nodeValue = leading + translate(original.trim()) + trailing;
    });
  }

  function applyI18n() {
    document.documentElement.lang = currentLanguage() === 'zh-CN' ? 'zh-CN' : 'en';
    translateTextNodes(document.body);
    document.querySelectorAll('[placeholder], [title]').forEach(el => {
      ['placeholder', 'title'].forEach(attr => {
        if (!el.hasAttribute(attr)) return;
        const key = `__i18n_${attr}`;
        const original = el[key] || el.getAttribute(attr);
        el[key] = original;
        el.setAttribute(attr, translate(original));
      });
    });
    const btn = document.getElementById('langBtn');
    if (btn) btn.textContent = currentLanguage() === 'zh-CN' ? 'EN' : '中';
  }

  function installLanguageButton() {
    if (document.getElementById('langBtn')) return;
    const themeBtn = document.getElementById('themeBtn');
    if (!themeBtn || !themeBtn.parentElement) return;
    const btn = document.createElement('button');
    btn.className = 'theme-toggle';
    btn.id = 'langBtn';
    btn.type = 'button';
    btn.title = currentLanguage() === 'zh-CN' ? 'Switch to English' : '切换到中文';
    btn.onclick = () => {
      localStorage.setItem(storageKey, currentLanguage() === 'zh-CN' ? 'en' : 'zh-CN');
      applyI18n();
      window.dispatchEvent(new CustomEvent('mcmon-language-change'));
    };
    themeBtn.parentElement.insertBefore(btn, themeBtn);
  }

  window.mcmonI18n = {
    lang: currentLanguage,
    t: translate,
    apply: applyI18n,
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      installLanguageButton();
      applyI18n();
    });
  } else {
    installLanguageButton();
    applyI18n();
  }
})();
