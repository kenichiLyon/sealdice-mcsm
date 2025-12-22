import { MCSMClient } from './client';

// Simple state manager to track pending logins per group
const loginState = new Map<string, string>(); // groupId -> alias

export function registerCommands(ext: seal.ExtInfo, client: MCSMClient) {
  const cmd = seal.ext.newCmdItemInfo();
  cmd.name = 'mcsm';
  cmd.help = `MCSM 管理指令:
.mcsm bind <alias> <proto_uuid> <core_uuid> - 绑定实例
.mcsm <start|stop|restart> <alias> - 管理实例
.mcsm status [alias] - 查看状态
.mcsm relogin <alias> - 扫码登录
.mcsm continue - 确认登录完成`;

  cmd.solve = (ctx, msg, args) => {
    const sub = args.getArgN(1);

    // Ensure connection
    client.connect(ctx);

    (async () => {
      try {
        switch (sub) {
          case 'bind':
            await handleBind(ctx, msg, args, client);
            break;
          case 'start':
          case 'stop':
          case 'restart':
          case 'fstop':
            await handleControl(ctx, msg, args, client, sub);
            break;
          case 'status':
            await handleStatus(ctx, msg, args, client);
            break;
          case 'relogin':
            await handleRelogin(ctx, msg, args, client);
            break;
          case 'continue':
            await handleContinue(ctx, msg, client);
            break;
          default:
            seal.replyToSender(ctx, msg, cmd.help);
        }
      } catch (e: any) {
        seal.replyToSender(ctx, msg, `执行失败: ${e.message}`);
      }
    })();

    return seal.ext.newCmdExecuteResult(true);
  };

  ext.cmdMap['mcsm'] = cmd;
}

async function handleBind(ctx: seal.MsgContext, msg: seal.Message, args: seal.CmdArgs, client: MCSMClient) {
  const alias = args.getArgN(2);
  const protoId = args.getArgN(3);
  const coreId = args.getArgN(4);
  if (!alias || !protoId || !coreId) {
    seal.replyToSender(ctx, msg, '用法: .mcsm bind <alias> <proto_uuid> <core_uuid>');
    return;
  }
  const res = await client.send('bind', { alias, protocol_id: protoId, core_id: coreId }, ctx);
  seal.replyToSender(ctx, msg, `绑定结果: ${res.code === 200 ? '成功' : res.message}`);
}

async function handleControl(ctx: seal.MsgContext, msg: seal.Message, args: seal.CmdArgs, client: MCSMClient, action: string) {
  const target = args.getArgN(2);
  const role = args.getArgN(3); // Optional
  if (!target) {
    seal.replyToSender(ctx, msg, `用法: .mcsm ${action} <alias> [role]`);
    return;
  }
  const params: Record<string, string> = { target };
  if (role) params['role'] = role;

  const res = await client.send(action, params, ctx);
  seal.replyToSender(ctx, msg, `指令发送: ${res.code === 200 ? '成功' : res.message}`);
}

async function handleStatus(ctx: seal.MsgContext, msg: seal.Message, args: seal.CmdArgs, client: MCSMClient) {
  const target = args.getArgN(2);
  const res = await client.send('status', target ? { target } : {}, ctx);

  if (res.code !== 200) {
    seal.replyToSender(ctx, msg, `查询失败: ${res.message}`);
    return;
  }

  let output = '';
  if (target) {
    // Instance Detail
    const d = res.data;
    output = `实例 ${target} 状态:\n运行: ${d.status === 1 ? '是' : '否'}\nCPU: ${d.process?.cpuUsage}%\n内存: ${d.process?.memory}`;
  } else {
    // Dashboard
    output = `MCSM 面板状态:\n版本: ${res.data.version}\n实例数: ${res.data.remoteCount?.total}`;
  }
  seal.replyToSender(ctx, msg, output);
}

async function handleRelogin(ctx: seal.MsgContext, msg: seal.Message, args: seal.CmdArgs, client: MCSMClient) {
  const target = args.getArgN(2);
  if (!target) {
    seal.replyToSender(ctx, msg, '用法: .mcsm relogin <alias>');
    return;
  }

  // Register state
  const groupId = ctx.group?.groupId || 'private';
  loginState.set(groupId, target);

  await client.send('relogin', { target }, ctx);
  seal.replyToSender(ctx, msg, '重登录流程已启动，请等待二维码...');
}

async function handleContinue(ctx: seal.MsgContext, msg: seal.Message, client: MCSMClient) {
  const groupId = ctx.group?.groupId || 'private';
  const alias = loginState.get(groupId);

  if (!alias) {
    seal.replyToSender(ctx, msg, '当前群没有进行中的重登录流程');
    return;
  }

  await client.send('continue', { target: alias }, ctx);
  seal.replyToSender(ctx, msg, '已发送继续指令');

  // Clear state after success? Or wait for success event?
  // Ideally clear it, but if it fails we might want to retry.
  // For now, keep it until next relogin overwrites or manually cleared (or timeout).
  // Let's remove it to keep it clean.
  loginState.delete(groupId);
}
