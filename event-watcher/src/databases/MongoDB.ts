import { ChainName, coalesceChainId } from '@certusone/wormhole-sdk/lib/cjs/utils/consts';
import BaseDB from './BaseDB';
import { VaaLog } from './types';
import * as mongoDB from 'mongodb';
import { env } from '../config';

const WORMHOLE_TX_COLLECTION: string = 'wormholeTx';
const WORMHOLE_LAST_BLOCK_COLLECTION: string = 'lastBlockByChain';

export default class MongoDB extends BaseDB {
  private client: mongoDB.MongoClient | null = null;
  private db: mongoDB.Db | null = null;
  private wormholeTxCollection: mongoDB.Collection | null = null;
  private lastTxBlockByChainCollection: mongoDB.Collection | null = null;

  constructor() {
    super();
  }

  async connect(): Promise<void> {
    try {
      this.client = new mongoDB.MongoClient(env.MONGODB_URI as string);
      this.db = this.client.db(env.MONGODB_DATABASE ?? 'wormhole');
      this.wormholeTxCollection = this.db.collection(WORMHOLE_TX_COLLECTION);
      this.lastTxBlockByChainCollection = this.db.collection(WORMHOLE_LAST_BLOCK_COLLECTION);
      await this.client?.connect();

      console.log('---CONNECTED TO MongoDB---');
    } catch (e) {
      throw new Error(`(MongoDB) Error: ${e}`);
    }
  }

  async getLastBlocksProcessed(): Promise<void> {
    const latestBlocks = await this.lastTxBlockByChainCollection?.findOne({});
    const json = JSON.parse(JSON.stringify(latestBlocks));
    this.lastBlockByChain = json || {};
  }

  override async storeVaaLogs(chain: ChainName, vaaLogs: VaaLog[]): Promise<void> {
    await this.wormholeTxCollection?.insertMany(vaaLogs);
  }

  override async storeLatestProcessBlock(chain: ChainName, lastBlock: number): Promise<void> {
    const chainId = coalesceChainId(chain);

    await this.lastTxBlockByChainCollection?.findOneAndUpdate(
      {},
      {
        $set: {
          [chainId]: lastBlock,
          updatedAt: new Date().getTime(),
        },
      },
      {
        upsert: true,
      },
    );
  }
}