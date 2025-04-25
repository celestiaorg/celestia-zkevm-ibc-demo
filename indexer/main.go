package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/berachain/beacon-kit/mod/consensus-types/pkg/types"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/version"
	client "github.com/celestiaorg/celestia-openrpc"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	"github.com/celestiaorg/celestia-openrpc/types/share"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	pb "github.com/rollkit/rollkit/types/pb/rollkit"
	bolt "go.etcd.io/bbolt"
)

const (
	dbPath        = "eth_celestia_mapping.db"
	bucketName    = "height_mappings"
	metaBucket    = "metadata"
	lastProcessed = "last_processed_height"
)

var (
	db         *bolt.DB
	shutdownCh = make(chan struct{})
	wg         sync.WaitGroup
)

// Config holds the application configuration
type Config struct {
	CelestiaNodeURL   string
	CelestiaAuthToken string
	CelestiaNamespace string
	APIPort           string
	HTTPTimeout       time.Duration
	ReconnectDelay    time.Duration
}

type InclusionHeightResponse struct {
	EthBlockNumber uint64 `json:"eth_block_number"`
	CelestiaHeight uint64 `json:"celestia_height"`
	BlobCommitment []byte `json:"blob_commitment"`
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	config := Config{
		CelestiaNodeURL:   getEnv("CELESTIA_NODE_URL", "ws://localhost:26658"),
		CelestiaAuthToken: getEnv("CELESTIA_NODE_AUTH_TOKEN", ""),
		CelestiaNamespace: getEnv("CELESTIA_NAMESPACE", "0f0f0f0f0f0f0f0f0f0f"),
		APIPort:           getEnv("API_PORT", "8080"),
		HTTPTimeout:       time.Duration(getEnvInt("HTTP_TIMEOUT_SECONDS", 30)) * time.Second,
		ReconnectDelay:    time.Duration(getEnvInt("RECONNECT_DELAY_SECONDS", 5)) * time.Second,
	}
	return config
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	strValue := os.Getenv(key)
	if strValue == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("Warning: Could not parse %s as integer, using default value %d", key, defaultValue)
		return defaultValue
	}
	return intValue
}

// setupDB initializes the BoltDB database
func setupDB() error {
	var err error
	db, err = bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("could not open db: %v", err)
	}

	// Create the buckets
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("could not create height mappings bucket: %v", err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte(metaBucket))
		if err != nil {
			return fmt.Errorf("could not create metadata bucket: %v", err)
		}
		return nil
	})

	return err
}

// storeMapping saves an Ethereum block number to Celestia height, blob index mapping
func storeMapping(ethBlockNum uint64, celestiaHeight uint64, blobCommitment []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// Convert ethBlockNum to bytes
		key := make([]byte, 8)
		binary.LittleEndian.PutUint64(key, ethBlockNum)

		// Convert celestiaHeight to bytes
		value := make([]byte, 8+len(blobCommitment))
		binary.LittleEndian.PutUint64(value[:8], celestiaHeight)
		copy(value[8:], blobCommitment)

		return b.Put(key, value)
	})
}

// updateLastProcessedHeight updates the last processed Celestia height
func updateLastProcessedHeight(height uint64) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metaBucket))
		value := strconv.FormatUint(height, 10)
		return b.Put([]byte(lastProcessed), []byte(value))
	})
}

// getLastProcessedHeight retrieves the last processed Celestia height
func getLastProcessedHeight() (uint64, error) {
	var height uint64 = 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metaBucket))
		v := b.Get([]byte(lastProcessed))

		if v == nil {
			return nil // No last processed height found
		}

		var err error
		height, err = strconv.ParseUint(string(v), 10, 64)
		return err
	})

	return height, err
}

// getMapping retrieves the Celestia block height and blob commitment for a given Ethereum block number
func getMapping(ethBlockNum uint64) (celestiaHeight uint64, blobCommitment []byte, isFound bool, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		// Convert ethBlockNum to bytes
		key := make([]byte, 8)
		binary.LittleEndian.PutUint64(key, ethBlockNum)

		v := b.Get(key)
		if v == nil {
			isFound = false
			return nil
		}

		isFound = true
		if len(v) < 8 {
			return errors.New("invalid value length")
		}
		celestiaHeight = binary.LittleEndian.Uint64(v[:8])
		blobCommitment = v[8:]
		return nil
	})
	return
}

// decodeRollkitBlock decodes a Rollkit block
func decodeRollkitBlock(blob []byte) (*pb.Block, error) {
	var pbBlock pb.Block
	err := proto.Unmarshal(blob, &pbBlock)
	if err != nil {
		return nil, err
	}
	return &pbBlock, nil
}

// decodeEthBlockNumber extracts the block number
func decodeEthBlockNumber(data []byte) (uint64, error) {
	block := &types.BeaconBlock{}
	block, err := block.NewFromSSZ(data, version.Deneb)
	if err != nil {
		return 0, err
	}
	return block.GetBody().GetExecutionPayload().GetNumber().Unwrap(), nil
}

// startIndexer starts the indexing service that listens for new Celestia blocks and extracts Ethereum block numbers
func startIndexer(ctx context.Context, config Config) {
	defer wg.Done()

	nsBytes, err := hex.DecodeString(config.CelestiaNamespace)
	if err != nil {
		panic(err)
	}

	namespace, err := share.NewBlobNamespaceV0(nsBytes)
	if err != nil {
		panic(err)
	}

	log.Println("Starting indexer service...")

	// Function to create and establish connection
	connectClient := func() (*client.Client, <-chan *header.ExtendedHeader, error) {
		c, err := client.NewClient(ctx, config.CelestiaNodeURL, config.CelestiaAuthToken)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create client: %v", err)
		}

		// Get the last processed height to start from
		lastHeight, err := getLastProcessedHeight()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get last processed height: %v", err)
		}

		// Subscribe to new headers
		headerChan, err := c.Header.Subscribe(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to subscribe: %v", err)
		}

		log.Printf("Connected to Celestia node, resuming from height %d", lastHeight)

		// If we have a last processed height, process any blocks we missed
		if lastHeight > 0 {
			go func() {
				// Get the latest height
				latestHeader, err := c.Header.GetByHeight(ctx, 0) // 0 = latest
				if err != nil {
					log.Printf("Error getting latest height: %v", err)
					return
				}

				latestHeight := latestHeader.Height()

				// Process any missed blocks (up to a reasonable limit)
				maxBackfill := uint64(100) // Limit how far back we'll go
				startHeight := lastHeight + 1

				if latestHeight-startHeight > maxBackfill {
					startHeight = latestHeight - maxBackfill
					log.Printf("Too many missed blocks, limiting backfill to %d blocks", maxBackfill)
				}

				log.Printf("Backfilling missed blocks from %d to %d", startHeight, latestHeight)

				for h := startHeight; h <= latestHeight; h++ {
					processHeight(ctx, c, namespace, h)
				}
			}()
		}

		return c, headerChan, nil
	}

	// Connect initially
	c, headerChan, err := connectClient()
	if err != nil {
		log.Printf("Initial connection failed: %v", err)
		// Don't quit on initial connection failure, the reconnect loop will try again
	}

	for {
		// If we're not connected, try to reconnect
		if c == nil {
			log.Printf("Attempting to reconnect in %v...", config.ReconnectDelay)
			select {
			case <-time.After(config.ReconnectDelay):
				var err error
				c, headerChan, err = connectClient()
				if err != nil {
					log.Printf("Reconnection failed: %v", err)
					continue
				}
			case <-shutdownCh:
				log.Println("Shutting down indexer...")
				return
			}
		}

		// Process headers
		select {
		case header, ok := <-headerChan:
			if !ok {
				log.Println("Header channel closed, will reconnect...")
				c = nil
				headerChan = nil
				continue
			}

			height := header.Height()
			log.Printf("Processing new block at height %d", height)

			// Process the height
			processHeight(ctx, c, namespace, height)

		case <-ctx.Done():
			log.Println("Context canceled, shutting down indexer...")
			return

		case <-shutdownCh:
			log.Println("Shutting down indexer...")
			return
		}
	}
}

// processHeight processes blobs at a specific Celestia height
func processHeight(ctx context.Context, c *client.Client, namespace share.Namespace, height uint64) {
	// Create a timeout context for this operation
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch all blobs at the specified height
	blobs, err := c.Blob.GetAll(timeoutCtx, height, []share.Namespace{namespace})
	if err != nil {
		log.Printf("Error fetching blobs at height %d: %v", height, err)
		return
	}

	log.Printf("Found %d blobs at height %d", len(blobs), height)

	for _, blob := range blobs {
		block, err := decodeRollkitBlock(blob.Blob.Data)
		if err != nil {
			log.Printf("Error decoding block at height %d: %v", height, err)
			continue
		}

		if len(block.Data.Txs) < 1 {
			log.Printf("No payload transaction found in block at height %d", height)
			continue
		}

		data := block.Data.Txs[0]
		ethBlockNum, err := decodeEthBlockNumber(data)
		if err != nil {
			log.Printf("Error decoding Ethereum block number at height %d: %v", height, err)
			continue
		}

		log.Printf("Found Ethereum block %d at Celestia height %d", ethBlockNum, height)

		// Store the mapping
		err = storeMapping(ethBlockNum, height, blob.Commitment)
		if err != nil {
			log.Printf("Error storing mapping: %v", err)
			continue
		}
	}

	// Update the last processed height
	err = updateLastProcessedHeight(height)
	if err != nil {
		log.Printf("Error updating last processed height: %v", err)
	}
}

// startAPI starts the HTTP API server
func startAPI(config Config) {
	defer wg.Done()

	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Get Celestia height for Ethereum block
	router.HandleFunc("/inclusion_height/{eth_block_number}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ethBlockNumStr := vars["eth_block_number"]

		// Parse Ethereum block number
		ethBlockNum64, err := strconv.ParseUint(ethBlockNumStr, 10, 16)
		if err != nil {
			http.Error(w, "Invalid Ethereum block number", http.StatusBadRequest)
			return
		}

		ethBlockNum := ethBlockNum64

		// Get mapping from database
		celestiaHeight, blobCommitment, found, err := getMapping(ethBlockNum)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		if !found {
			http.Error(w, "Ethereum block not found", http.StatusNotFound)
			return
		}

		// Return the Celestia height
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := InclusionHeightResponse{
			EthBlockNumber: ethBlockNum,
			CelestiaHeight: celestiaHeight,
			BlobCommitment: blobCommitment,
		}

		// Marshal to JSON and write the response
		jsonData, err := json.Marshal(response)
		if err != nil {
			// Handle error
			http.Error(w, "Failed to generate response", http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	}).Methods("GET")

	// Status endpoint
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		lastHeight, err := getLastProcessedHeight()
		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"last_processed_celestia_height": %d}`, lastHeight)
	}).Methods("GET")

	// Start the server
	server := &http.Server{
		Addr:         ":" + config.APIPort,
		Handler:      router,
		ReadTimeout:  config.HTTPTimeout,
		WriteTimeout: config.HTTPTimeout,
	}

	// Handle graceful shutdown
	go func() {
		<-shutdownCh
		log.Println("Shutting down API server...")

		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()

	log.Printf("API server listening on port %s", config.APIPort)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("API server error: %v", err)
		return
	}
}

func main() {
	// Load configuration
	config := loadConfig()

	// Setup the database
	if err := setupDB(); err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer db.Close()

	// Set up context for the indexer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize shutdown channel
	shutdownCh = make(chan struct{})

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v", sig)
		close(shutdownCh)
		cancel()
	}()

	// Start the API service
	wg.Add(1)
	go startAPI(config)
	log.Println("API service started")

	// Start the indexer service
	wg.Add(1)
	go startIndexer(ctx, config)
	log.Println("Indexer service started")

	// Add a ready signal
	log.Println("All services started successfully, running...")

	// Wait for services to finish
	wg.Wait()
	log.Println("All services have shut down. Exiting.")
}
