import { WSMessage, Request, Response, RequestContext, PushEvent } from './types';

export class MCSMClient {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string;
  private pendingRequests = new Map<string, (res: Response) => void>();
  private sessionStore = new Map<string, RequestContext>();
  private reconnectTimer: any = null;
  private heartbeatTimer: any = null;
  private isConnected = false;
  private ctx: seal.MsgContext | null = null;

  constructor(url: string, token: string) {
    this.url = url;
    this.token = token;
  }

  public updateConfig(url: string, token: string) {
    this.url = url;
    this.token = token;
    if (this.isConnected) {
      this.disconnect();
      this.connect();
    }
  }

  public registerContext(req_id: string, ctx: seal.MsgContext) {
    // Assuming ctx has group id info.
    // In Sealdice JS, ctx.group.id is not directly exposed as string sometimes, but let's try.
    // Actually, ctx.Group.GroupId or similar.
    // If not available, we store the whole ctx object to reply.
    this.sessionStore.set(req_id, {
      source_ctx: ctx,
      group_id: ctx.group?.groupId || '',
      timestamp: Date.now()
    });

    // Cleanup old sessions > 10 mins
    const now = Date.now();
    for (const [k, v] of this.sessionStore) {
      if (now - v.timestamp > 600000) {
        this.sessionStore.delete(k);
      }
    }
  }

  public connect(ctx?: seal.MsgContext) {
    if (this.isConnected || this.ws) return;
    if (ctx) this.ctx = ctx;

    if (typeof (globalThis as any).WebSocket === 'undefined') {
      if (this.ctx) seal.replyToSender(this.ctx, seal.newMessage(), '当前环境不支持 WebSocket');
      return;
    }

    try {
      // Append token to URL if present
      let wsUrl = this.url;
      if (this.token) {
        const separator = wsUrl.includes('?') ? '&' : '?';
        wsUrl += `${separator}token=${encodeURIComponent(this.token)}`;
      }
      this.ws = new (globalThis as any).WebSocket(wsUrl);
    } catch (e) {
      console.error('WS Creation Failed:', e);
      this.scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      this.isConnected = true;
      console.log('MCSM Bridge Connected');
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
      this.startHeartbeat();
    };

    this.ws.onclose = () => {
      this.isConnected = false;
      this.ws = null;
      console.log('MCSM Bridge Disconnected');
      this.stopHeartbeat();
      this.scheduleReconnect();
    };

    this.ws.onerror = (err) => {
      console.error('MCSM Bridge Error:', err);
      this.isConnected = false;
    };

    this.ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data) as WSMessage;
        this.handleMessage(msg);
      } catch (e) {
        console.error('JSON Parse Error:', e);
      }
    };
  }

  public disconnect() {
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
      this.isConnected = false;
      this.stopHeartbeat();
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, 5000);
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      if (this.ws && this.isConnected) {
        // Send a ping frame if supported, or text ping
        // Goja WebSocket might not support 'ping' method directly?
        // Let's send a custom ping message.
        try {
           this.ws.send(JSON.stringify({ type: 'ping' }));
        } catch(e) {}
      }
    }, 30000);
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private handleMessage(msg: WSMessage) {
    if (msg.type === 'response') {
      const resp = msg as Response;
      const cb = this.pendingRequests.get(resp.req_id);
      if (cb) {
        this.pendingRequests.delete(resp.req_id);
        cb(resp);
      }
    } else if (msg.type === 'event') {
      this.handleEvent(msg as PushEvent);
    } else if (msg.type === 'error') {
      console.error('Server Error:', msg);
    }
  }

  private handleEvent(msg: PushEvent) {
    let ctx = this.ctx; // Default fallback
    if (msg.req_id) {
      const session = this.sessionStore.get(msg.req_id);
      if (session) {
        ctx = session.source_ctx;
      }
    }

    if (!ctx) return;

    if (msg.event === 'qrcode') {
      const data = msg.data;
      if (data.url) {
        seal.replyToSender(ctx, seal.newMessage(), `[MCSM] 请扫描二维码登录 (Alias: ${data.alias})\n[CQ:image,file=${data.url}]`);
      }
    } else if (msg.event === 'log') {
      if (typeof msg.data === 'string') {
         seal.replyToSender(ctx, seal.newMessage(), `[MCSM Log] ${msg.data}`);
      }
    } else if (msg.event === 'success') {
      seal.replyToSender(ctx, seal.newMessage(), `[MCSM] ${msg.data}`);
    } else if (msg.event === 'error') {
      const d = msg.data as any;
      seal.replyToSender(ctx, seal.newMessage(), `[MCSM Error] ${d.msg || d}`);
    }
  }

  public async send(command: string, params: Record<string, string>, ctx?: seal.MsgContext): Promise<Response> {
    if (!this.isConnected || !this.ws) {
      throw new Error('WebSocket未连接');
    }

    const req_id = this.uuid();

    // Register context if provided
    if (ctx) {
      this.registerContext(req_id, ctx);
    }

    const payload = {
      action: command,
      command: command,
      req_id,
      params
    };

    return new Promise<Response>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(req_id);
        this.sessionStore.delete(req_id); // Clean up session on timeout
        reject(new Error('请求超时'));
      }, 60000);

      this.pendingRequests.set(req_id, (res) => {
        clearTimeout(timeout);
        resolve(res);
      });

      this.ws!.send(JSON.stringify(payload));
    });
  }

  private uuid(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }
}
