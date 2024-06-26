package service

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wormhole-foundation/wormhole-explorer/common/client/alert"
	"github.com/wormhole-foundation/wormhole-explorer/common/configuration"
	"github.com/wormhole-foundation/wormhole-explorer/common/dbutil"
	"github.com/wormhole-foundation/wormhole-explorer/common/domain"
	"github.com/wormhole-foundation/wormhole-explorer/common/health"
	"github.com/wormhole-foundation/wormhole-explorer/common/logger"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/builder"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/config"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/http/infrastructure"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/http/redeem"
	cwAlert "github.com/wormhole-foundation/wormhole-explorer/contract-watcher/internal/alert"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/internal/ankr"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/internal/metrics"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/processor"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/storage"
	"github.com/wormhole-foundation/wormhole-explorer/contract-watcher/watcher"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
)

type exitCode int

func handleExit() {
	if r := recover(); r != nil {
		if e, ok := r.(exitCode); ok {
			os.Exit(int(e))
		}
		panic(r) // not an Exit, bubble up
	}
}

type watchersConfig struct {
	ankr        []config.WatcherBlockchainAddresses
	avalanche   *config.WatcherBlockchainAddresses
	base        *config.WatcherBlockchainAddresses
	baseSepolia *config.WatcherBlockchainAddresses
	ethereum    *config.WatcherBlockchainAddresses
	celo        *config.WatcherBlockchainAddresses
	moonbeam    *config.WatcherBlockchainAddresses
	polygon     *config.WatcherBlockchainAddresses
	terra       *config.WatcherBlockchain
	rateLimit   rateLimitConfig
}

type rateLimitConfig struct {
	ankr        int
	avalanche   int
	base        int
	baseSepolia int
	celo        int
	ethereum    int
	moonbeam    int
	polygon     int
	terra       int
}

func Run() {

	defer handleExit()
	rootCtx, rootCtxCancel := context.WithCancel(context.Background())

	cfg, err := configuration.LoadFromEnv[config.ServiceConfiguration](rootCtx)
	if err != nil {
		log.Fatal("Error creating config", err)
	}

	var testnetConfig *config.TestnetConfiguration
	if configuration.IsTestnet(cfg.P2pNetwork) {
		testnetConfig, err = configuration.LoadFromEnv[config.TestnetConfiguration](rootCtx)
		if err != nil {
			log.Fatal("Error loading testnet rpc config: ", err)
		}
	}

	logger := logger.New("wormhole-explorer-contract-watcher", logger.WithLevel(cfg.LogLevel))

	logger.Info("Starting wormhole-explorer-contract-watcher ...")

	//setup DB connection
	db, err := dbutil.Connect(rootCtx, logger, cfg.MongoURI, cfg.MongoDatabase, false)
	if err != nil {
		logger.Fatal("failed to connect MongoDB", zap.Error(err))
	}

	// get health check functions.
	healthChecks, err := newHealthChecks(rootCtx, db.Database)
	if err != nil {
		logger.Fatal("failed to create health checks", zap.Error(err))
	}

	// create metrics client
	metrics := metrics.NewPrometheusMetrics(cfg.Environment)

	// create alert client
	alerts := newAlertClient(cfg, logger)

	// create repositories
	repo := storage.NewRepository(db.Database, metrics, alerts, logger)

	// create watchers
	watchers := newWatchers(cfg, testnetConfig, repo, metrics, logger)

	//create processor
	processor := processor.NewProcessor(watchers, logger)
	processor.Start(rootCtx)

	// create and start server.
	redeemController := redeem.NewController(watchers, logger)
	server := infrastructure.NewServer(logger, cfg.Port, cfg.PprofEnabled, redeemController, healthChecks...)
	server.Start()

	logger.Info("Started wormhole-explorer-contract-watcher")

	// Waiting for signal
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-rootCtx.Done():
		logger.Warn("Terminating with root context cancelled.")
	case signal := <-sigterm:
		logger.Info("Terminating with signal.", zap.String("signal", signal.String()))
	}

	logger.Info("root context cancelled, exiting...")
	rootCtxCancel()

	logger.Info("Closing processor ...")
	processor.Close()

	logger.Info("closing MongoDB connection...")
	db.DisconnectWithTimeout(10 * time.Second)

	logger.Info("Closing Http server ...")
	server.Stop()

	logger.Info("Finished wormhole-explorer-contract-watcher")
}

func newHealthChecks(_ context.Context, db *mongo.Database) ([]health.Check, error) {
	return []health.Check{health.Mongo(db)}, nil
}

func newWatchers(config *config.ServiceConfiguration, testnetConfig *config.TestnetConfiguration, repo *storage.Repository, metrics metrics.Metrics, logger *zap.Logger) []watcher.ContractWatcher {
	var watchers *watchersConfig
	switch config.P2pNetwork {
	case domain.P2pMainNet:
		watchers = newWatchersForMainnet(config)
	case domain.P2pTestNet:
		watchers = newWatchersForTestnet(config, testnetConfig)
	default:
		watchers = &watchersConfig{}
	}

	result := make([]watcher.ContractWatcher, 0)

	// add evm watchers
	evmLimiter := ratelimit.New(watchers.rateLimit.ankr, ratelimit.Per(time.Second))
	ankrClient := ankr.NewAnkrSDK(config.AnkrUrl, evmLimiter, metrics)
	for _, w := range watchers.ankr {
		params := watcher.EVMParams{ChainID: w.ChainID, Blockchain: w.Name, SizeBlocks: w.SizeBlocks,
			WaitSeconds: w.WaitSeconds, InitialBlock: w.InitialBlock, MethodsByAddress: w.MethodsByAddress}
		result = append(result, watcher.NewEVMWatcher(ankrClient, repo, params, metrics, logger))
	}

	// add ethereum watcher
	if watchers.ethereum != nil {
		ethereumWatcher := builder.CreateEvmWatcher(watchers.rateLimit.ethereum, config.EthereumUrl, *watchers.ethereum, logger, repo, metrics)
		result = append(result, ethereumWatcher)
	}

	// add avalanche watcher
	if watchers.avalanche != nil {
		avalancheWatcher := builder.CreateEvmWatcher(watchers.rateLimit.avalanche, config.AvalancheUrl, *watchers.avalanche, logger, repo, metrics)
		result = append(result, avalancheWatcher)
	}

	// add terra watcher
	if watchers.terra != nil {
		terraWatcher := builder.CreateTerraWatcher(watchers.rateLimit.terra, config.TerraUrl, *watchers.terra, logger, repo, metrics)
		result = append(result, terraWatcher)
	}

	// add moonbeam watcher
	if watchers.moonbeam != nil {
		moonbeamWatcher := builder.CreateEvmWatcher(watchers.rateLimit.moonbeam, config.MoonbeamUrl, *watchers.moonbeam, logger, repo, metrics)
		result = append(result, moonbeamWatcher)
	}

	// add celo watcher
	if watchers.celo != nil {
		celoWatcher := builder.CreateEvmWatcher(watchers.rateLimit.celo, config.CeloUrl, *watchers.celo, logger, repo, metrics)
		result = append(result, celoWatcher)
	}

	// add base watcher
	if watchers.base != nil {
		baseWatcher := builder.CreateEvmWatcher(watchers.rateLimit.base, config.BaseUrl, *watchers.base, logger, repo, metrics)
		result = append(result, baseWatcher)
	}

	// add base sepolia watcher
	if watchers.baseSepolia != nil {
		baseSepoliaWatcher := builder.CreateEvmWatcher(watchers.rateLimit.baseSepolia, testnetConfig.BaseSepoliaBaseUrl, *watchers.baseSepolia, logger, repo, metrics)
		result = append(result, baseSepoliaWatcher)
	}

	// add polygon watcher
	if watchers.polygon != nil {
		polygonWatcher := builder.CreateEvmWatcher(watchers.rateLimit.polygon, config.PolygonUrl, *watchers.polygon, logger, repo, metrics)
		result = append(result, polygonWatcher)
	}

	return result
}

func newWatchersForMainnet(cfg *config.ServiceConfiguration) *watchersConfig {
	return &watchersConfig{
		ankr: []config.WatcherBlockchainAddresses{
			config.BSC_MAINNET,
			config.FANTOM_MAINNET,
		},
		avalanche: &config.AVALANCHE_MAINNET,
		base:      &config.BASE_MAINNET,
		celo:      &config.CELO_MAINNET,
		ethereum:  &config.ETHEREUM_MAINNET,
		moonbeam:  &config.MOONBEAM_MAINNET,
		polygon:   &config.POLYGON_MAINNET,
		terra:     &config.TERRA_MAINNET,

		rateLimit: rateLimitConfig{
			ankr:      cfg.AnkrRequestsPerSecond,
			avalanche: cfg.AvalancheRequestsPerSecond,
			base:      cfg.BaseRequestsPerSecond,
			celo:      cfg.CeloRequestsPerSecond,
			ethereum:  cfg.EthereumRequestsPerSecond,
			moonbeam:  cfg.MoonbeamRequestsPerSecond,
			polygon:   cfg.PolygonRequestsPerSecond,
			terra:     cfg.TerraRequestsPerSecond,
		},
	}
}

func newWatchersForTestnet(cfg *config.ServiceConfiguration, testnetCfg *config.TestnetConfiguration) *watchersConfig {
	return &watchersConfig{
		ankr: []config.WatcherBlockchainAddresses{
			config.BSC_TESTNET,
			config.FANTOM_TESTNET,
		},
		avalanche:   &config.AVALANCHE_TESTNET,
		celo:        &config.CELO_TESTNET,
		base:        &config.BASE_TESTNET,
		baseSepolia: &config.BASE_SEPOLIA_TESTNET,
		ethereum:    &config.ETHEREUM_TESTNET,
		moonbeam:    &config.MOONBEAM_TESTNET,
		polygon:     &config.POLYGON_TESTNET,
		rateLimit: rateLimitConfig{
			ankr:        cfg.AnkrRequestsPerSecond,
			avalanche:   cfg.AvalancheRequestsPerSecond,
			base:        cfg.BaseRequestsPerSecond,
			baseSepolia: testnetCfg.BaseSepoliaRequestsPerMinute,
			celo:        cfg.CeloRequestsPerSecond,
			ethereum:    cfg.EthereumRequestsPerSecond,
			moonbeam:    cfg.MoonbeamRequestsPerSecond,
			polygon:     cfg.PolygonRequestsPerSecond,
			terra:       cfg.TerraRequestsPerSecond,
		},
	}
}

func newAlertClient(config *config.ServiceConfiguration, logger *zap.Logger) alert.AlertClient {

	if !config.AlertEnabled {
		return alert.NewDummyClient()
	}
	alertConfig := alert.AlertConfig{
		Environment: config.Environment,
		ApiKey:      config.AlertApiKey,
		Enabled:     config.AlertEnabled,
	}
	client, err := alert.NewAlertService(alertConfig, cwAlert.LoadAlerts)
	if err != nil {
		logger.Fatal("Error creating alert client", zap.Error(err))
	}
	return client
}
