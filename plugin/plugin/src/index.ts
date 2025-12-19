function main() {
  let ext = seal.ext.find('mcsm-remote');
  if (!ext) {
    ext = seal.ext.new('mcsm-remote', 'sealdice-mcsm', '0.1.0');
    seal.ext.register(ext);
  }

  seal.ext.registerStringConfig(ext, 'ws_url', 'ws://127.0.0.1:8088/ws', 'Server WebSocket 地址');
  seal.ext.registerStringConfig(ext, 'token', '', '鉴权令牌（可空）');

  let ws: any = null;
  let wsReady = false;
  const pendingRequests = new Map<string, (res: any) => void>();

  function uuid(): string {
    const s4 = () => Math.floor((1 + Math.random()) * 0x10000).toString(16).substring(1);
    return `${s4()}${s4()}-${s4()}-${s4()}-${s4()}-${s4()}${s4()}${s4()}`;
  }

  function ensureWS(ctx: seal.MsgContext) {
    if (ws && wsReady) return;
    const url = seal.ext.getStringConfig(ext, 'ws_url');
    if (typeof (globalThis as any).WebSocket === 'undefined') {
      seal.replyToSender(ctx, seal.newMessage(), '当前环境不支持 WebSocket');
      return;
    }
    ws = new (globalThis as any).WebSocket(url);
    ws.onopen = () => { wsReady = true; };
    ws.onclose = () => { wsReady = false; ws = null; };
    ws.onerror = () => { wsReady = false; };
    ws.onmessage = (ev: any) => {
      try {
        const msg = JSON.parse(ev.data);
        if (msg.type === 'response') {
          const cb = pendingRequests.get(msg.request_id);
          if (cb) { pendingRequests.delete(msg.request_id); cb(msg); }
        } else if (msg.type === 'event') {
          if (msg.event === 'qrcode_ready') {
            const data = msg.data;
            const img = seal.base64ToImage(data.qrcode);
            seal.replyToSender(ctx, seal.newMessage(), `[二维码] alias=${data.alias} 生成于 ${data.generated_at}\n${img}`);
          }
        }
      } catch {}
    };
  }

  async function sendCommand(ctx: seal.MsgContext, cmd: string, params: Record<string, string>) {
    ensureWS(ctx);
    if (!ws || !wsReady) {
      seal.replyToSender(ctx, seal.newMessage(), 'WS未连接，请稍后重试');
      return;
    }
    const request_id = uuid();
    const payload = { request_id, command: cmd, params };
    return new Promise<void>((resolve) => {
      pendingRequests.set(request_id, (res) => {
        if (res.code === 200) {
          if (res.data && typeof res.data === 'string') {
            seal.replyToSender(ctx, seal.newMessage(), res.data);
          } else {
            seal.replyToSender(ctx, seal.newMessage(), JSON.stringify(res.data));
          }
        } else {
          seal.replyToSender(ctx, seal.newMessage(), `错误：${res.message || JSON.stringify(res.data)}`);
        }
        resolve();
      });
      ws.send(JSON.stringify(payload));
    });
  }

  function cmd(name: string, help: string, handler: (ctx: seal.MsgContext, msg: seal.Message, args: seal.CmdArgs) => Promise<void>) {
    const c = seal.ext.newCmdItemInfo();
    c.name = name;
    c.help = help;
    c.solve = (ctx, msg, cmdArgs) => {
      handler(ctx, msg, cmdArgs);
      return seal.ext.newCmdExecuteResult(true);
    };
    ext.cmdMap[name] = c;
  }

  cmd('bind', '绑定别名：.bind <alias> <instance_id>', async (ctx, msg, args) => {
    const alias = args.getArgN(1);
    const id = args.getArgN(2);
    if (!alias || !id) {
      seal.replyToSender(ctx, msg, '用法：.bind <alias> <instance_id>');
      return;
    }
    await sendCommand(ctx, 'bind', { alias, instance_id: id });
  });

  const controlHelp = '控制实例：.start/.stop/.restart/.fstop <alias|id>';
  for (const name of ['start','stop','restart','fstop']) {
    cmd(name, controlHelp, async (ctx, msg, args) => {
      const target = args.getArgN(1);
      if (!target) {
        seal.replyToSender(ctx, msg, `用法：.${name} <alias|id>`);
        return;
      }
      await sendCommand(ctx, name, { target });
    });
  }

  cmd('status', '查询：.status [alias|id]', async (ctx, msg, args) => {
    const target = args.getArgN(1);
    await sendCommand(ctx, 'status', target ? { target } : {});
  });

  cmd('relogin', '重登录流程：.relogin <alias>', async (ctx, msg, args) => {
    const target = args.getArgN(1);
    if (!target) {
      seal.replyToSender(ctx, msg, '用法：.relogin <alias>');
      return;
    }
    await sendCommand(ctx, 'relogin', { target });
  });

  cmd('continue', '继续登录：.continue <alias>', async (ctx, msg, args) => {
    const target = args.getArgN(1);
    if (!target) {
      seal.replyToSender(ctx, msg, '用法：.continue <alias>');
      return;
    }
    await sendCommand(ctx, 'continue', { target });
  });
}

main();
