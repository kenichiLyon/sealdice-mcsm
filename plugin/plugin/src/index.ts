import { MCSMClient } from './client';
import { registerCommands } from './commands';

function main() {
  let ext = seal.ext.find('mcsm-bridge');
  if (!ext) {
    ext = seal.ext.new('mcsm-bridge', 'sealdice', '1.0.0');
    seal.ext.register(ext);
  }

  seal.ext.registerStringConfig(ext, 'ws_url', 'ws://127.0.0.1:8088/ws', 'Server WebSocket 地址');
  seal.ext.registerStringConfig(ext, 'token', '', '鉴权令牌');

  const url = seal.ext.getStringConfig(ext, 'ws_url');
  const token = seal.ext.getStringConfig(ext, 'token');

  const client = new MCSMClient(url, token);
  
  // Register commands
  registerCommands(ext, client);
  
  // Attempt initial connection if configured
  if (url) {
    client.connect();
  }
}

main();
