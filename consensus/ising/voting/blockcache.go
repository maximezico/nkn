package voting

import (
	"sync"
	"time"

	. "github.com/nknorg/nkn/common"
	"github.com/nknorg/nkn/core/ledger"
	"github.com/nknorg/nkn/util/log"
)

const (
	MaxCachedBlocks = 1000
)

// BlockInfo hold the block which waiting for voting
type BlockInfo struct {
	block    *ledger.Block
	lifetime *time.Timer // expired block will be auto removed
}

type BlockCache struct {
	sync.RWMutex
	cap           int
	currentHeight uint32                  //current block height
	cache         map[uint32][]*BlockInfo // height and block mapping
	hashes        map[Uint256]uint32      // for block fast searching
}

// When receive new block message from consensus layer, cache it.
func NewCache() *BlockCache {
	blockCache := &BlockCache{
		cap:           MaxCachedBlocks,
		currentHeight: 0,
		cache:         make(map[uint32][]*BlockInfo),
		hashes:        make(map[Uint256]uint32),
	}
	go blockCache.Cleanup()

	return blockCache
}

// BlockInCache returns whether the block has been cached.
func (bc *BlockCache) BlockInCache(hash Uint256) bool {
	bc.RLock()
	defer bc.RUnlock()

	if _, ok := bc.hashes[hash]; ok {
		return true
	}

	return false
}

// GetBlockFromCache returns block according to block hash passed in.
func (bc *BlockCache) GetBlockFromCache(hash Uint256) *ledger.Block {
	bc.RLock()
	defer bc.RUnlock()

	if i, ok := bc.hashes[hash]; !ok {
		return nil
	} else {
		for _, v := range bc.cache[i] {
			if hash.CompareTo(v.block.Hash()) == 0 {
				return v.block
			}
		}
		return nil
	}
}

// GetBestBlockFromCache returns latest block in cache
func (bc *BlockCache) GetBestBlockFromCache() *ledger.Block {
	bc.RLock()
	defer bc.RUnlock()

	if blockInfos, ok := bc.cache[bc.currentHeight]; ok {
		if blockInfos == nil {
			return nil
		}
		minBlock := blockInfos[0]
		minBlockHash := minBlock.block.Hash()
		for _, v := range blockInfos[1:] {
			if minBlockHash.CompareTo(v.block.Hash()) == 1 {
				minBlock = v
				minBlockHash = v.block.Hash()
			}
		}
		return minBlock.block
	}

	return nil
}

// GetBestBlockFromCache returns latest block in cache
func (bc *BlockCache) GetWorseBlockFromCache() *ledger.Block {
	bc.RLock()
	defer bc.RUnlock()

	if blockInfos, ok := bc.cache[bc.currentHeight]; ok {
		if blockInfos == nil {
			return nil
		}
		minBlock := blockInfos[0]
		minBlockHash := minBlock.block.Hash()
		for _, v := range blockInfos[1:] {
			if minBlockHash.CompareTo(v.block.Hash()) == -1 {
				minBlock = v
				minBlockHash = v.block.Hash()
			}
		}
		return minBlock.block
	}

	return nil
}

// RemoveBlockFromCache return true if the block doesn't exist in cache.
func (bc *BlockCache) RemoveBlockFromCache(hash Uint256) error {
	bc.Lock()
	defer bc.Unlock()

	if bc.BlockInCache(hash) {
		height := bc.hashes[hash]
		delete(bc.hashes, hash)

		var blockInfos []*BlockInfo
		for k, v := range bc.cache[height] {
			if hash.CompareTo(v.block.Hash()) == 0 {
				blockInfos = append(blockInfos, bc.cache[height][:k]...)
				blockInfos = append(blockInfos, bc.cache[height][k+1:]...)
				break
			}
		}
		if blockInfos == nil {
			delete(bc.cache, height)
			// if the last block of current height is being removed,
			// decrease current height together
			if height == bc.currentHeight {
				bc.currentHeight--
			}
		} else {
			bc.cache[height] = blockInfos
		}
	}

	return nil
}

// CachedBlockNum return the block number in cache
func (bc *BlockCache) CachedBlockNum() int {
	bc.RLock()
	defer bc.RUnlock()

	count := 0
	for _, v := range bc.cache {
		count += len(v)
	}

	return count
}

// AddBlockToCache returns nil if block already existed in cache
func (bc *BlockCache) AddBlockToCache(block *ledger.Block) error {
	hash := block.Hash()
	if bc.BlockInCache(hash) {
		return nil
	}
	bc.Lock()
	defer bc.Unlock()
	// TODO FIFO cleanup, if cap space is not enough then
	// remove block from cache according to FIFO
	blockInfo := &BlockInfo{
		block:    block,
		lifetime: time.NewTimer(time.Hour),
	}
	blockHeight := block.Header.Height
	bc.cache[blockHeight] = append(bc.cache[blockHeight], blockInfo)
	bc.hashes[hash] = blockHeight
	if blockHeight > bc.currentHeight {
		bc.currentHeight = blockHeight
	}

	return nil
}

func (bc *BlockCache) Dump() {
	log.Infof("current height: %d", bc.currentHeight)
	for height, blockInfos := range bc.cache {
		log.Infof("\t height: %d", height)
		for _, v := range blockInfos {
			hash := v.block.Hash()
			log.Infof("\t\t hash: %s", BytesToHexString(hash.ToArray()))
		}
	}
}

// Cleanup is a background routine used for cleaning up expired block in cache
func (bc *BlockCache) Cleanup() {
	ticket := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticket.C:
			for _, blockInfos := range bc.cache {
				for _, info := range blockInfos {
					select {
					case <-info.lifetime.C:
						bc.RemoveBlockFromCache(info.block.Hash())
					}
				}
			}
		}
	}
}