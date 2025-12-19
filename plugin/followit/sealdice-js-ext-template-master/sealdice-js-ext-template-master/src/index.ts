import { sample } from "lodash-es";
import { nameList } from "./utils";

function main() {
  let ext = seal.ext.find('test');
  if (!ext) {
    ext = seal.ext.new('test', '木落', '1.0.0');
    seal.ext.register(ext);
  }

  const cmdSeal = seal.ext.newCmdItemInfo();
  cmdSeal.name = 'seal';
  cmdSeal.help = '召唤一只海豹，可用.seal <名字> 命名';

  cmdSeal.solve = (ctx, msg, cmdArgs) => {
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        if (!val) val = sample(nameList);
        seal.replyToSender(ctx, msg, `你抓到一只海豹！取名为${val}\n它的逃跑意愿为${Math.ceil(Math.random() * 100)}`);
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  }

  ext.cmdMap['seal'] = cmdSeal;
}

main();
