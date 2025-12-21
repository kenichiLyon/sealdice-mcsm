import { WSMessage, Request, Response } from './types';

export class MCSMClient {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string;
  private pendingRequests = new Map<string, (res: Response) => void>();
  private reconnectTimer: any = null;
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

  public connect(ctx?: seal.MsgContext) {
    if (this.isConnected || this.ws) return;
    if (ctx) this.ctx = ctx;

    if (typeof (globalThis as any).WebSocket === 'undefined') {
      if (this.ctx) seal.replyToSender(this.ctx, seal.newMessage(), '当前环境不支持 WebSocket');
      return;
    }

    try {
      this.ws = new (globalThis as any).WebSocket(this.url);
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
    };

    this.ws.onclose = () => {
      this.isConnected = false;
      this.ws = null;
      console.log('MCSM Bridge Disconnected');
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
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, 5000);
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
      // Handle events (like QR code)
      this.handleEvent(msg as any);
    } else if (msg.type === 'error') {
      // Global error handling or log
      console.error('Server Error:', msg);
    }
  }

  private handleEvent(msg: { event: string, data: any }) {
    if (msg.event === 'qrcode_ready' && this.ctx) {
      const data = msg.data;
      const img = seal.base64ToImage(data.qrcode);
      seal.replyToSender(this.ctx, seal.newMessage(), `[MCSM] 请扫描二维码登录 (Alias: ${data.alias})\n生成时间: ${data.generated_at}\n${img}`);
    }
  }

  public async send(command: string, params: Record<string, string>): Promise<Response> {
    if (!this.isConnected || !this.ws) {
      throw new Error('WebSocket未连接');
    }

    const req_id = this.uuid();
    const payload = {
      action: command, // Note: Server expects "action" or "command"? Previous code sent "command". User protocol says "action".
      // Let's check server code: ws/server.go reads "Command". 
      // Protocol description says "action". I should align. 
      // Server struct Request has `Command string`. 
      // I will send "command" to match server, but user spec said "action". 
      // Wait, user spec: "Request: { "action": "string", ... }". 
      // Server code: `type Request struct { Command string ... }`. 
      // I should update server or plugin. Since I am refactoring server too, I will use "action" in server to match spec.
      // For now, I'll send BOTH or check. 
      // Let's assume I will fix server to use "action".
      action: command,
      command: command, // Backwards compatibility for now until server is updated
      req_id,
      params
    };

    return new Promise<Response>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(req_id);
        reject(new Error('请求超时'));
      }, 10000);

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
