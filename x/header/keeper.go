package header

import (
	"context"

	"cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	// binaryCodec is used to marshal and unmarshal data from the store.
	binaryCodec codec.BinaryCodec

	// storeKey is key that is used to fetch the signal store from the multi
	// store.
	storeService store.KVStoreService
}

func NewKeeper(
	binaryCodec codec.BinaryCodec, storeService store.KVStoreService,
) Keeper {
	return Keeper{
		binaryCodec:  binaryCodec,
		storeService: storeService,
	}
}

func (k Keeper) BeginBlocker(ctx sdk.Context) error {
	// Prune headers that are older than the retention period
	err := k.PruneHeaders(ctx)
	if err != nil {
		return err
	}

	// Save the block header
	height := ctx.BlockHeight()
	headerHash := ctx.HeaderHash()
	err = k.SaveHeaderHash(ctx, height, headerHash)
	if err != nil {
		return err
	}
	return nil
}

func (k Keeper) SaveHeaderHash(ctx context.Context, height int64, headerHash []byte) error {
	store := k.storeService.OpenKVStore(ctx)
	err := store.Set(sdk.Uint64ToBigEndian(uint64(height)), headerHash)
	return err
}

func (k Keeper) GetHeaderHash(ctx context.Context, height int64) ([]byte, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := sdk.Uint64ToBigEndian(uint64(height))
	key, err := store.Get(key)
	if err != nil {
		return nil, false
	}
	headerHash, err := store.Get(key)
	if err != nil {
		return nil, false
	}
	return headerHash, true
}

// PruneHeaders prunes block headers that are older than the retention window.
func (k Keeper) PruneHeaders(ctx sdk.Context) error {
	store := k.storeService.OpenKVStore(ctx)
	iterator, err := store.Iterator(nil, nil) // Start from the lowest key
	if err != nil {
		panic(err)
	}
	defer iterator.Close()

	latestHeight, ok := k.GetLatestSavedBlockHeight(ctx)
	if !ok {
		return nil
	}

	// Calculate the minimum height to retain
	minHeightToRetain := latestHeight - retentionPeriod

	for ; iterator.Valid(); iterator.Next() {
		// Convert the key (height) from []byte to int64
		height := sdk.BigEndianToUint64(iterator.Key())

		// If the height is below the minimum height to retain, delete it
		if height < minHeightToRetain {
			err := store.Delete(iterator.Key())
			if err != nil {
				return err
			}
		} else {
			// Since entries are sorted by height, we can break early
			break
		}
	}
	return nil
}

func (k Keeper) GetLatestSavedBlockHeight(ctx context.Context) (uint64, bool) {
	store := k.storeService.OpenKVStore(ctx)
	storeIterator, err := store.ReverseIterator(nil, nil)
	if err != nil {
		panic(err)
	}
	defer storeIterator.Close()

	if !storeIterator.Valid() {
		return 0, false
	}
	// parse the key to get the height
	height := sdk.BigEndianToUint64(storeIterator.Key())
	return height, true
}
