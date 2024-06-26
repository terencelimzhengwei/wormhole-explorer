package txhash

import (
	"context"
	"fmt"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole-explorer/common/domain"
	"github.com/wormhole-foundation/wormhole-explorer/fly/deduplicator"
	"go.uber.org/zap"
)

type dedupTxHashStore struct {
	txHashStore  TxHashStore
	deduplicator *deduplicator.Deduplicator
	logger       *zap.Logger
}

func NewDedupTxHashStore(txHashStore TxHashStore, deduplicator *deduplicator.Deduplicator, logger *zap.Logger) *dedupTxHashStore {
	return &dedupTxHashStore{
		txHashStore:  txHashStore,
		deduplicator: deduplicator,
		logger:       logger,
	}
}

func (d *dedupTxHashStore) Set(ctx context.Context, uniqueVaaID string, txHash TxHash) error {
	key := fmt.Sprintf("observation:%s", uniqueVaaID)
	return d.deduplicator.Apply(ctx, key, func() error {
		return d.txHashStore.Set(ctx, uniqueVaaID, txHash)
	})
}

func (d *dedupTxHashStore) SetObservation(ctx context.Context, o *gossipv1.SignedObservation) error {
	txHash, err := CreateTxHash(d.logger, o)
	if err != nil {
		d.logger.Error("Error creating txHash", zap.Error(err))
		return err
	}
	uniqueVaaID := domain.CreateUniqueVaaIDByObservation(o)
	return d.Set(ctx, uniqueVaaID, *txHash)
}

func (d *dedupTxHashStore) Get(ctx context.Context, uniqueVaaID string) (*string, error) {
	return d.txHashStore.Get(ctx, uniqueVaaID)
}

func (r *dedupTxHashStore) GetName() string {
	return "dedup"
}
