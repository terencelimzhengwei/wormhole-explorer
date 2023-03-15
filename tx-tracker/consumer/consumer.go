package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/wormhole-foundation/wormhole-explorer/parser/parser"

	"github.com/wormhole-foundation/wormhole-explorer/txtracker/chains"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/config"
	"github.com/wormhole-foundation/wormhole-explorer/txtracker/queue"
	sdk "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// SourceTxStatus is meant to be a user-facing enum that describes the status of the source transaction.
type SourceTxStatus string

const (
	// SourceTxStatusChainNotSupported indicates that the processing failed due to the chain ID not being supported.
	//
	// (i.e.: there is no adapter for that chain yet)
	SourceTxStatusChainNotSupported SourceTxStatus = "chainNotSupported"

	// SourceTxStatusInternalError represents an internal, unspecified error.
	SourceTxStatusInternalError SourceTxStatus = "internalError"

	// SourceTxStatusConfirmed indicates that the transaciton has been processed successfully.
	SourceTxStatusConfirmed SourceTxStatus = "confirmed"
)

const (
	numRetries = 2
	retryDelay = 5 * time.Second
)

const AppIdPortalTokenBridge = "PORTAL_TOKEN_BRIDGE"

// Consumer consumer struct definition.
type Consumer struct {
	consumeFunc        queue.VAAConsumeFunc
	cfg                *config.Settings
	logger             *zap.Logger
	globalTransactions *mongo.Collection
	vaaPayloadParser   parser.ParserVAAAPIClient
}

// New creates a new vaa consumer.
func New(
	consumeFunc queue.VAAConsumeFunc,
	cfg *config.Settings,
	logger *zap.Logger,
	db *mongo.Database,
) (*Consumer, error) {

	vaaPayloadParser, err := parser.NewParserVAAAPIClient(
		cfg.VaaPayloadParserTimeout,
		cfg.VaaPayloadParserUrl,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VAA parser client: %w", err)
	}

	c := Consumer{
		consumeFunc:        consumeFunc,
		cfg:                cfg,
		logger:             logger,
		globalTransactions: db.Collection("globalTransactions"),
		vaaPayloadParser:   vaaPayloadParser,
	}

	return &c, nil
}

// Start consumes messages from VAA queue, parse and store those messages in a repository.
func (c *Consumer) Start(ctx context.Context) {
	go func() {
		for msg := range c.consumeFunc(ctx) {
			event := msg.Data()

			// Check if message is expired.
			if msg.IsExpired() {
				c.logger.Warn("Message with VAA expired", zap.String("id", event.ID))
				msg.Failed()
				continue
			}

			// Do not process messages from PythNet
			if event.ChainID == sdk.ChainIDPythNet {
				msg.Done()
				continue
			}

			// Parse the VAA's payload
			parsedPayload, err := c.vaaPayloadParser.Parse(
				uint16(event.ChainID),
				event.EmitterAddress,
				event.Sequence,
				event.Vaa,
			)
			if err == parser.ErrNotFound {
				c.logger.Debug("Skipping message - no parsed registered for this (chain, emitter) pair",
					zap.String("vaaId", event.ID),
				)
				msg.Done()
				continue
			}
			if err != nil {
				c.logger.Error("Failed to parse VAA payload",
					zap.String("vaaId", event.ID),
					zap.Error(err),
				)
				msg.Done()
				continue
			}

			// Skip messages that have not been generated by the portal token bridge
			if parsedPayload.AppID != AppIdPortalTokenBridge {
				c.logger.Debug("Skipping VAA because it was not generated by the portal token bridge",
					zap.String("vaaId", event.ID),
				)
				msg.Done()
				continue
			}

			// Get transaction details from the emitter blockchain
			//
			// If the transaction is not found, will retry a few times before giving up.
			var txStatus SourceTxStatus
			var txDetail *chains.TxDetail
			for attempts := numRetries; attempts > 0; attempts-- {

				txDetail, err = chains.FetchTx(ctx, c.cfg, event.ChainID, event.TxHash)

				switch {
				// If the transaction is not found, retry after a delay
				case err == chains.ErrTransactionNotFound:
					txStatus = SourceTxStatusInternalError
					time.Sleep(retryDelay)
					continue

				// If the chain ID is not supported, give up
				case err == chains.ErrChainNotSupported:
					c.logger.Debug("Failed to fetch source transaction details - chain not supported",
						zap.String("vaaId", event.ID),
					)
					txStatus = SourceTxStatusChainNotSupported
					break

				// If there is an internal error, give up
				case err != nil:
					c.logger.Error("Failed to fetch source transaction details",
						zap.String("vaaId", event.ID),
						zap.Error(err),
					)
					txStatus = SourceTxStatusInternalError
					break

				// Success
				case err == nil:
					txStatus = SourceTxStatusConfirmed
					break
				}
			}

			// Store source transaction details in the database
			err = updateSourceTxData(ctx, c.globalTransactions, event, txDetail, txStatus)
			if err != nil {
				c.logger.Error("Failed to upsert source transaction details",
					zap.String("vaaId", event.ID),
					zap.Error(err),
				)
			} else {
				c.logger.Debug("Successfuly updated source transaction details in the database",
					zap.String("id", event.ID),
					zap.Any("details", txDetail),
				)
			}

			msg.Done()
		}
	}()
}

func updateSourceTxData(
	ctx context.Context,
	vaas *mongo.Collection,
	event *queue.VaaEvent,
	txDetail *chains.TxDetail,
	txStatus SourceTxStatus,
) error {

	fields := bson.D{
		{Key: "chainId", Value: event.ChainID},
		{Key: "txHash", Value: event.TxHash},
		{Key: "status", Value: txStatus},
	}

	if txDetail != nil {
		fields = append(fields, primitive.E{Key: "timestamp", Value: txDetail.Timestamp})
		fields = append(fields, primitive.E{Key: "signer", Value: txDetail.Signer})

		// It is still to be defined whether we want to expose this field to the API consumers,
		// since it can be obtained from the original TxHash.
		//fields = append(fields, primitive.E{Key: "nativeTxHash", Value: txDetail.NativeTxHash})
	}

	update := bson.D{
		{
			Key: "$set",
			Value: bson.D{
				{
					Key:   "originTx",
					Value: fields,
				},
			},
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := vaas.UpdateByID(ctx, event.ID, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert source tx information: %w", err)
	}

	return nil
}