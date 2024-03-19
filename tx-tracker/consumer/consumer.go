package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/wormhole-foundation/wormhole-explorer/common/pool"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/chains"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/internal/metrics"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/queue"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	sdk "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Consumer consumer struct definition.
type Consumer struct {
	consumeFunc queue.ConsumeFunc
	rpcpool     map[vaa.ChainID]*pool.Pool
	logger      *zap.Logger
	repository  *Repository
	metrics     metrics.Metrics
	p2pNetwork  string
	workersSize int
}

// New creates a new vaa consumer.
func New(
	consumeFunc queue.ConsumeFunc,
	rpcPool map[vaa.ChainID]*pool.Pool,
	ctx context.Context,
	logger *zap.Logger,
	repository *Repository,
	metrics metrics.Metrics,
	p2pNetwork string,
	workersSize int,
) *Consumer {

	c := Consumer{
		consumeFunc: consumeFunc,
		rpcpool:     rpcPool,
		logger:      logger,
		repository:  repository,
		metrics:     metrics,
		p2pNetwork:  p2pNetwork,
		workersSize: workersSize,
	}

	return &c
}

// Start consumes messages from VAA queue, parse and store those messages in a repository.
func (c *Consumer) Start(ctx context.Context) {
	ch := c.consumeFunc(ctx)
	for i := 0; i < c.workersSize; i++ {
		go c.producerLoop(ctx, ch)
	}
}

func (c *Consumer) producerLoop(ctx context.Context, ch <-chan queue.ConsumerMessage) {

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			c.logger.Debug("Received message", zap.String("vaaId", msg.Data().ID), zap.String("trackId", msg.Data().TrackID))
			switch msg.Data().Type {
			case queue.SourceChainEvent:
				c.processSourceTx(ctx, msg)
			case queue.TargetChainEvent:
				c.processTargetTx(ctx, msg)
			default:
				c.logger.Error("Unknown message type", zap.String("trackId", msg.Data().TrackID), zap.Any("type", msg.Data().Type))
			}
		}
	}
}

func (c *Consumer) processSourceTx(ctx context.Context, msg queue.ConsumerMessage) {

	event := msg.Data()

	// Do not process messages from PythNet
	if event.ChainID == sdk.ChainIDPythNet {
		msg.Done()
		c.logger.Debug("Skipping pythNet message", zap.String("trackId", event.TrackID), zap.String("vaaId", event.ID))
		return
	}

	if event.ChainID == sdk.ChainIDNear {
		msg.Done()
		c.logger.Warn("Skipping vaa from near", zap.String("trackId", event.TrackID), zap.String("vaaId", event.ID))
		return
	}

	start := time.Now()

	c.metrics.IncVaaUnfiltered(uint16(event.ChainID))

	// Process the VAA
	p := ProcessSourceTxParams{
		TrackID:   event.TrackID,
		Timestamp: event.Timestamp,
		VaaId:     event.ID,
		ChainId:   event.ChainID,
		Emitter:   event.EmitterAddress,
		Sequence:  event.Sequence,
		TxHash:    event.TxHash,
		Metrics:   c.metrics,
		Overwrite: false, // avoid processing the same transaction twice
	}
	_, err := ProcessSourceTx(ctx, c.logger, c.rpcpool, c.repository, &p, c.p2pNetwork)

	// add vaa processing duration metrics
	c.metrics.AddVaaProcessedDuration(uint16(event.ChainID), time.Since(start).Seconds())

	elapsedLog := zap.Uint64("elapsedTime", uint64(time.Since(start).Milliseconds()))
	// Log a message informing the processing status
	if errors.Is(err, chains.ErrChainNotSupported) {
		msg.Done()
		c.logger.Info("Skipping VAA - chain not supported",
			zap.String("trackId", event.TrackID),
			zap.String("vaaId", event.ID),
			elapsedLog,
		)
	} else if errors.Is(err, ErrAlreadyProcessed) {
		msg.Done()
		c.logger.Warn("Origin message already processed - skipping",
			zap.String("trackId", event.TrackID),
			zap.String("vaaId", event.ID),
			elapsedLog,
		)
	} else if err != nil {
		msg.Failed()
		c.logger.Error("Failed to process originTx",
			zap.String("trackId", event.TrackID),
			zap.String("vaaId", event.ID),
			zap.Error(err),
			elapsedLog,
		)
	} else {
		msg.Done()
		c.logger.Info("Origin transaction processed successfully",
			zap.String("trackId", event.TrackID),
			zap.String("id", event.ID),
			elapsedLog,
		)
		c.metrics.IncOriginTxInserted(uint16(event.ChainID))
	}
}

func (c *Consumer) processTargetTx(ctx context.Context, msg queue.ConsumerMessage) {

	event := msg.Data()

	attr, ok := queue.GetAttributes[*queue.TargetChainAttributes](event)
	if !ok || attr == nil {
		msg.Failed()
		c.logger.Error("Failed to get attributes from message", zap.String("trackId", event.TrackID), zap.String("vaaId", event.ID))
		return
	}
	start := time.Now()

	// Process the VAA
	p := ProcessTargetTxParams{
		TrackID:        event.TrackID,
		VaaId:          event.ID,
		ChainId:        event.ChainID,
		Emitter:        event.EmitterAddress,
		TxHash:         event.TxHash,
		BlockTimestamp: event.Timestamp,
		BlockHeight:    attr.BlockHeight,
		Method:         attr.Method,
		From:           attr.From,
		To:             attr.To,
		Status:         attr.Status,
	}
	err := ProcessTargetTx(ctx, c.logger, c.repository, &p)

	elapsedLog := zap.Uint64("elapsedTime", uint64(time.Since(start).Milliseconds()))
	if err != nil {
		msg.Failed()
		c.logger.Error("Failed to process destinationTx",
			zap.String("trackId", event.TrackID),
			zap.String("vaaId", event.ID),
			zap.Error(err),
			elapsedLog,
		)
	} else {
		msg.Done()
		c.logger.Info("Destination transaction processed successfully",
			zap.String("trackId", event.TrackID),
			zap.String("id", event.ID),
			elapsedLog,
		)
	}
}
