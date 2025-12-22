export interface Request {
  action: string;
  req_id: string;
  params: Record<string, string>;
}

export interface Response<T = any> {
  req_id: string;
  type: 'response' | 'push' | 'error';
  data: T;
  code?: number;
  message?: string;
}

export interface EventData {
  alias: string;
  generated_at?: string;
  qrcode?: string;
  url?: string;
  msg?: string;
}

export interface PushEvent {
  type: 'event';
  req_id?: string; // Optional: if server sends it
  event: string;
  data: EventData;
}

export interface RequestContext {
  source_ctx: any; // seal.MsgContext is native object, treat as any or strict type if available
  group_id: string;
  timestamp: number;
}

export interface ErrorResponse {
  type: 'error';
  code: string;
  message: string;
  req_id?: string;
}

export type WSMessage = Response | PushEvent | ErrorResponse;
