import {
  ChainName,
  CosmWasmChainName,
  EVMChainName,
} from '@certusone/wormhole-sdk/lib/cjs/utils/consts';
import { AlgorandWatcher } from './AlgorandWatcher';
import { AptosWatcher } from './AptosWatcher';
import { ArbitrumWatcher } from './ArbitrumWatcher';
import { BSCWatcher } from './BSCWatcher';
import { CosmwasmWatcher } from './CosmwasmWatcher';
import { EVMWatcher } from './EVMWatcher';
import { InjectiveExplorerWatcher } from './InjectiveExplorerWatcher';
import { MoonbeamWatcher } from './MoonbeamWatcher';
import { NearWatcher } from './NearWatcher';
import { PolygonWatcher } from './PolygonWatcher';
import { SolanaWatcher } from './SolanaWatcher';
import { TerraExplorerWatcher } from './TerraExplorerWatcher';
import { SuiWatcher } from './SuiWatcher';
import { WatcherOptionTypes } from './types';
import { SeiExplorerWatcher } from './SeiExplorerWatcher';

export function makeFinalizedWatcher(chainName: ChainName): WatcherOptionTypes {
  if (chainName === 'solana') {
    return new SolanaWatcher();
  } else if (['ethereum', 'karura', 'acala'].includes(chainName)) {
    return new EVMWatcher(chainName as EVMChainName, 'finalized');
  } else if (chainName === 'bsc') {
    return new BSCWatcher();
  } else if (chainName === 'polygon') {
    return new PolygonWatcher();
  } else if (
    ['avalanche', 'oasis', 'fantom', 'klaytn', 'celo', 'optimism', 'base'].includes(chainName)
  ) {
    return new EVMWatcher(chainName as EVMChainName);
  } else if (chainName === 'algorand') {
    return new AlgorandWatcher();
  } else if (chainName === 'moonbeam') {
    return new MoonbeamWatcher();
  } else if (chainName === 'arbitrum') {
    return new ArbitrumWatcher();
  } else if (chainName === 'aptos') {
    return new AptosWatcher();
  } else if (chainName === 'near') {
    return new NearWatcher();
  } else if (chainName === 'injective') {
    return new InjectiveExplorerWatcher();
  } else if (chainName === 'terra') {
    return new TerraExplorerWatcher('terra');
  } else if (['terra2', 'xpla'].includes(chainName)) {
    return new CosmwasmWatcher(chainName as CosmWasmChainName);
  } else if (chainName === 'sui') {
    return new SuiWatcher();
  } else if (chainName === 'sei') {
    return new SeiExplorerWatcher();
  } else {
    throw new Error(`Attempted to create finalized watcher for unsupported chain ${chainName}`);
  }
}
