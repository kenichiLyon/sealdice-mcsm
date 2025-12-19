function main() {
  let ext = seal.ext.find('mcsm');
  if (!ext) {
    ext = seal.ext.new('mcsm', 'Sealdice-MCSM', '0.2.0');
    seal.ext.registerStringConfig(ext, 'serverUrl', 'http://localhost:3000', 'MCSM 中间件地址');
    seal.ext.register(ext);
  }

  const cmdMcsm = seal.ext.newCmdItemInfo();
  cmdMcsm.name = 'mcsm';
  cmdMcsm.help =
    '.mcsm bind <名称> <daemonId> <instanceUuid>\n' +
    '.mcsm unbind <名称>\n' +
    '.mcsm status\n' +
    '.mcsm relogin <名称>';

  const enableWS = true
  let ws: WebSocket | null = null
  let retry = 0
  const connectWS = () => {
    const url = seal.ext.getStringConfig(ext, 'serverUrl').replace(/\/+$/, '') + '/ws'
    try {
      ws = new WebSocket(url)
      ws.onopen = () => {
        retry = 0
      }
      ws.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data as any)
          if (data && data.type === 'heartbeat') {
            return
          }
        } catch {}
      }
      ws.onclose = () => {
        retry++
        const delay = Math.min(30000, 2000 * retry)
        setTimeout(connectWS, delay)
      }
      ws.onerror = () => {
        try { ws && ws.close() } catch {}
      }
    } catch {}
  }
  if (enableWS) {
    connectWS()
  }

  cmdMcsm.solve = (ctx, msg, cmdArgs) => {
    const serverUrl = seal.ext.getStringConfig(ext, 'serverUrl');
    const op = cmdArgs.getArgN(1);
    if (!op || op === 'help') {
      const ret = seal.ext.newCmdExecuteResult(true);
      ret.showHelp = true;
      return ret;
    }
    const contextId =
      msg.messageType === 'group' && ctx.group && ctx.group.groupId
        ? `g:${ctx.group.groupId}`
        : ctx.player && ctx.player.userId
        ? `u:${ctx.player.userId}`
        : `u:${msg.sender?.userId || 'unknown'}`;
    if (op === 'bind') {
      const name = cmdArgs.getArgN(2);
      const daemonId = cmdArgs.getArgN(3);
      const uuid = cmdArgs.getArgN(4);
      if (!name || !daemonId || !uuid) {
        seal.replyToSender(ctx, msg, '用法: .mcsm bind <名称> <daemonId> <instanceUuid>');
        const ret = seal.ext.newCmdExecuteResult(true);
        return ret;
      }
      fetch(`${serverUrl}/bind`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contextId, instanceName: name, uuid, daemonId }),
      })
        .then((r) => r.json().then((data) => ({ r, data })).catch(() => ({ r, data: null })))
        .then(({ r, data }) => {
          if (r.status >= 200 && r.status < 300 && data && data.success) {
            seal.replyToSender(ctx, msg, `已绑定 ${name}`);
          } else {
            seal.replyToSender(ctx, msg, `绑定失败 ${r.status}`);
          }
        });
      return seal.ext.newCmdExecuteResult(true);
    }
    if (op === 'unbind') {
      const name = cmdArgs.getArgN(2);
      if (!name) {
        seal.replyToSender(ctx, msg, '用法: .mcsm unbind <名称>');
        const ret = seal.ext.newCmdExecuteResult(true);
        return ret;
      }
      fetch(`${serverUrl}/unbind`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contextId, instanceName: name }),
      })
        .then((r) => r.json().then((data) => ({ r, data })).catch(() => ({ r, data: null })))
        .then(({ r, data }) => {
          if (r.status >= 200 && r.status < 300 && data && data.success) {
            seal.replyToSender(ctx, msg, `已解绑 ${name}`);
          } else {
            seal.replyToSender(ctx, msg, `解绑失败 ${r.status}`);
          }
        });
      return seal.ext.newCmdExecuteResult(true);
    }
    if (op === 'status') {
      fetch(`${serverUrl}/status?contextId=${encodeURIComponent(contextId)}`)
        .then((r) => r.json().then((data) => ({ r, data })).catch(() => ({ r, data: null })))
        .then(({ r, data }) => {
          if (!(r.status >= 200 && r.status < 300) || !data || !data.success) {
            seal.replyToSender(ctx, msg, `查询失败 ${r.status}`);
            return;
          }
          const list = data.data || [];
          if (!list.length) {
            seal.replyToSender(ctx, msg, '无绑定实例');
            return;
          }
          const lines = list.map((i: any) => {
            const st = typeof i.status === 'number' ? i.status : i.process && i.process.running ? 1 : 0;
            return `${i.name || '-'} (${i.uuid}): ${st}`;
          });
          seal.replyToSender(ctx, msg, lines.join('\n'));
        });
      return seal.ext.newCmdExecuteResult(true);
    }
    if (op === 'relogin') {
      const name = cmdArgs.getArgN(2);
      if (!name) {
        seal.replyToSender(ctx, msg, '用法: .mcsm relogin <名称>');
        const ret = seal.ext.newCmdExecuteResult(true);
        return ret;
      }
      fetch(`${serverUrl}/relogin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contextId, instanceName: name }),
      })
        .then((r) => r.json().then((data) => ({ r, data })).catch(() => ({ r, data: null })))
        .then(({ r, data }) => {
          if (!(r.status >= 200 && r.status < 300) || !data || !data.success) {
            seal.replyToSender(ctx, msg, `重启失败 ${r.status}`);
            return;
          }
          if (data.qrCode) {
            seal.replyToSender(ctx, msg, `登录链接或二维码: ${data.qrCode}`);
          } else {
            seal.replyToSender(ctx, msg, '已重启，未捕获到二维码');
          }
        });
      return seal.ext.newCmdExecuteResult(true);
    }
    const ret = seal.ext.newCmdExecuteResult(true);
    ret.showHelp = true;
    return ret;
  };

  ext.cmdMap['mcsm'] = cmdMcsm;
}

main();
