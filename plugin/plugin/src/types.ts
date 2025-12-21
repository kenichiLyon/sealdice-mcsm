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
  generated_at: string;
  qrcode: string;
}

export interface PushEvent {
  type: 'event';
  event: string;
  data: any;
}

export interface ErrorResponse {
  type: 'error';
  code: string;
  message: string;
}

export type WSMessage = Response | PushEvent | ErrorResponse;
