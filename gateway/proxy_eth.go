package gateway

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
)

func (gw *Node) EthAccounts(ctx context.Context) ([]ethtypes.EthAddress, error) {
	// gateway provides public API, so it can't hold user accounts
	return []ethtypes.EthAddress{}, nil
}

func (gw *Node) EthBlockNumber(ctx context.Context) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	return gw.target.EthBlockNumber(ctx)
}

func (gw *Node) EthGetBlockTransactionCountByNumber(ctx context.Context, blkNum ethtypes.EthUint64) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	head, err := gw.target.ChainHead(ctx)
	if err != nil {
		return 0, err
	}
	if err := gw.checkTipsetHeight(head, abi.ChainEpoch(blkNum)); err != nil {
		return 0, err
	}

	return gw.target.EthGetBlockTransactionCountByNumber(ctx, blkNum)
}

func (gw *Node) tskByEthHash(ctx context.Context, blkHash ethtypes.EthHash) (types.TipSetKey, error) {
	tskCid := blkHash.ToCid()
	tskBlk, err := gw.target.ChainReadObj(ctx, tskCid)
	if err != nil {
		return types.EmptyTSK, err
	}
	tsk := new(types.TipSetKey)
	if err := tsk.UnmarshalCBOR(bytes.NewReader(tskBlk)); err != nil {
		return types.EmptyTSK, xerrors.Errorf("cannot unmarshal block into tipset key: %w", err)
	}

	return *tsk, nil
}

func (gw *Node) checkBlkHash(ctx context.Context, blkHash ethtypes.EthHash) error {
	tsk, err := gw.tskByEthHash(ctx, blkHash)
	if err != nil {
		return err
	}

	return gw.checkTipsetKey(ctx, tsk)
}

func (gw *Node) checkBlkParam(ctx context.Context, blkParam string) error {
	if blkParam == "earliest" {
		// also not supported in node impl
		return fmt.Errorf("block param \"earliest\" is not supported")
	}

	switch blkParam {
	case "pending", "latest":
		// those will be recent enough, so we don't need to check
		return nil
	default:
		var num ethtypes.EthUint64
		err := num.UnmarshalJSON([]byte(`"` + blkParam + `"`))
		if err != nil {
			return fmt.Errorf("cannot parse block number: %v", err)
		}
		head, err := gw.target.ChainHead(ctx)
		if err != nil {
			return err
		}
		if err := gw.checkTipsetHeight(head, abi.ChainEpoch(num)); err != nil {
			return err
		}
	}

	return nil
}

func (gw *Node) EthGetBlockTransactionCountByHash(ctx context.Context, blkHash ethtypes.EthHash) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	if err := gw.checkBlkHash(ctx, blkHash); err != nil {
		return 0, err
	}

	return gw.target.EthGetBlockTransactionCountByHash(ctx, blkHash)
}

func (gw *Node) EthGetBlockByHash(ctx context.Context, blkHash ethtypes.EthHash, fullTxInfo bool) (ethtypes.EthBlock, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthBlock{}, err
	}

	if err := gw.checkBlkHash(ctx, blkHash); err != nil {
		return ethtypes.EthBlock{}, err
	}

	return gw.target.EthGetBlockByHash(ctx, blkHash, fullTxInfo)
}

func (gw *Node) EthGetBlockByNumber(ctx context.Context, blkNum string, fullTxInfo bool) (ethtypes.EthBlock, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthBlock{}, err
	}

	if err := gw.checkBlkParam(ctx, blkNum); err != nil {
		return ethtypes.EthBlock{}, err
	}

	return gw.target.EthGetBlockByNumber(ctx, blkNum, fullTxInfo)
}

func (gw *Node) EthGetTransactionByHash(ctx context.Context, txHash *ethtypes.EthHash) (*ethtypes.EthTx, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	return gw.target.EthGetTransactionByHash(ctx, txHash)
}

func (gw *Node) EthGetTransactionCount(ctx context.Context, sender ethtypes.EthAddress, blkOpt string) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	if err := gw.checkBlkParam(ctx, blkOpt); err != nil {
		return 0, err
	}

	return gw.target.EthGetTransactionCount(ctx, sender, blkOpt)
}

func (gw *Node) EthGetTransactionReceipt(ctx context.Context, txHash ethtypes.EthHash) (*api.EthTxReceipt, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	return gw.target.EthGetTransactionReceipt(ctx, txHash)
}

func (gw *Node) EthGetTransactionByBlockHashAndIndex(ctx context.Context, blkHash ethtypes.EthHash, txIndex ethtypes.EthUint64) (ethtypes.EthTx, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthTx{}, err
	}

	if err := gw.checkBlkHash(ctx, blkHash); err != nil {
		return ethtypes.EthTx{}, err
	}

	return gw.target.EthGetTransactionByBlockHashAndIndex(ctx, blkHash, txIndex)
}

func (gw *Node) EthGetTransactionByBlockNumberAndIndex(ctx context.Context, blkNum ethtypes.EthUint64, txIndex ethtypes.EthUint64) (ethtypes.EthTx, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthTx{}, err
	}

	head, err := gw.target.ChainHead(ctx)
	if err != nil {
		return ethtypes.EthTx{}, err
	}
	if err := gw.checkTipsetHeight(head, abi.ChainEpoch(blkNum)); err != nil {
		return ethtypes.EthTx{}, err
	}

	return gw.target.EthGetTransactionByBlockNumberAndIndex(ctx, blkNum, txIndex)
}

func (gw *Node) EthGetCode(ctx context.Context, address ethtypes.EthAddress, blkOpt string) (ethtypes.EthBytes, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	if err := gw.checkBlkParam(ctx, blkOpt); err != nil {
		return nil, err
	}

	return gw.target.EthGetCode(ctx, address, blkOpt)
}

func (gw *Node) EthGetStorageAt(ctx context.Context, address ethtypes.EthAddress, position ethtypes.EthBytes, blkParam string) (ethtypes.EthBytes, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	if err := gw.checkBlkParam(ctx, blkParam); err != nil {
		return nil, err
	}

	return gw.target.EthGetStorageAt(ctx, address, position, blkParam)
}

func (gw *Node) EthGetBalance(ctx context.Context, address ethtypes.EthAddress, blkParam string) (ethtypes.EthBigInt, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthBigInt(big.Zero()), err
	}

	if err := gw.checkBlkParam(ctx, blkParam); err != nil {
		return ethtypes.EthBigInt(big.Zero()), err
	}

	return gw.target.EthGetBalance(ctx, address, blkParam)
}

func (gw *Node) EthChainId(ctx context.Context) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	return gw.target.EthChainId(ctx)
}

func (gw *Node) NetVersion(ctx context.Context) (string, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return "", err
	}

	return gw.target.NetVersion(ctx)
}

func (gw *Node) NetListening(ctx context.Context) (bool, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return false, err
	}

	return gw.target.NetListening(ctx)
}

func (gw *Node) EthProtocolVersion(ctx context.Context) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	return gw.target.EthProtocolVersion(ctx)
}

func (gw *Node) EthGasPrice(ctx context.Context) (ethtypes.EthBigInt, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthBigInt(big.Zero()), err
	}

	return gw.target.EthGasPrice(ctx)
}

var EthFeeHistoryMaxBlockCount = 128 // this seems to be expensive; todo: figure out what is a good number that works with everything

func (gw *Node) EthFeeHistory(ctx context.Context, blkCount ethtypes.EthUint64, newestBlk string, rewardPercentiles []float64) (ethtypes.EthFeeHistory, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthFeeHistory{}, err
	}

	if err := gw.checkBlkParam(ctx, newestBlk); err != nil {
		return ethtypes.EthFeeHistory{}, err
	}

	if blkCount > ethtypes.EthUint64(EthFeeHistoryMaxBlockCount) {
		return ethtypes.EthFeeHistory{}, fmt.Errorf("block count too high")
	}

	return gw.target.EthFeeHistory(ctx, blkCount, newestBlk, rewardPercentiles)
}

func (gw *Node) EthMaxPriorityFeePerGas(ctx context.Context) (ethtypes.EthBigInt, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthBigInt(big.Zero()), err
	}

	return gw.target.EthMaxPriorityFeePerGas(ctx)
}

func (gw *Node) EthEstimateGas(ctx context.Context, tx ethtypes.EthCall) (ethtypes.EthUint64, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return 0, err
	}

	// todo limit gas? to what?
	return gw.target.EthEstimateGas(ctx, tx)
}

func (gw *Node) EthCall(ctx context.Context, tx ethtypes.EthCall, blkParam string) (ethtypes.EthBytes, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	if err := gw.checkBlkParam(ctx, blkParam); err != nil {
		return nil, err
	}

	// todo limit gas? to what?
	return gw.target.EthCall(ctx, tx, blkParam)
}

func (gw *Node) EthSendRawTransaction(ctx context.Context, rawTx ethtypes.EthBytes) (ethtypes.EthHash, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthHash{}, err
	}

	return gw.target.EthSendRawTransaction(ctx, rawTx)
}

func (gw *Node) EthGetLogs(ctx context.Context, filter *ethtypes.EthFilterSpec) (*ethtypes.EthFilterResult, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	if filter.FromBlock != nil {
		if err := gw.checkBlkParam(ctx, *filter.FromBlock); err != nil {
			return nil, err
		}
	}
	if filter.ToBlock != nil {
		if err := gw.checkBlkParam(ctx, *filter.ToBlock); err != nil {
			return nil, err
		}
	}
	if filter.BlockHash != nil {
		if err := gw.checkBlkHash(ctx, *filter.BlockHash); err != nil {
			return nil, err
		}
	}

	return gw.target.EthGetLogs(ctx, filter)
}

/* FILTERS: Those are stateful.. figure out how to properly either bind them to users, or time out? */

func (gw *Node) EthGetFilterChanges(ctx context.Context, id ethtypes.EthFilterID) (*ethtypes.EthFilterResult, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	ft := filterTrackerFromContext(ctx)
	ft.lk.Lock()
	_, ok := ft.userFilters[id]
	ft.lk.Unlock()

	if !ok {
		return nil, nil
	}

	return gw.target.EthGetFilterChanges(ctx, id)
}

func (gw *Node) EthGetFilterLogs(ctx context.Context, id ethtypes.EthFilterID) (*ethtypes.EthFilterResult, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return nil, err
	}

	ft := filterTrackerFromContext(ctx)
	ft.lk.Lock()
	_, ok := ft.userFilters[id]
	ft.lk.Unlock()

	if !ok {
		return nil, nil
	}

	return gw.target.EthGetFilterLogs(ctx, id)
}

func (gw *Node) EthNewFilter(ctx context.Context, filter *ethtypes.EthFilterSpec) (ethtypes.EthFilterID, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthFilterID{}, err
	}

	return addUserFilterLimited(ctx, func() (ethtypes.EthFilterID, error) {
		return gw.target.EthNewFilter(ctx, filter)
	})
}

func (gw *Node) EthNewBlockFilter(ctx context.Context) (ethtypes.EthFilterID, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthFilterID{}, err
	}

	return addUserFilterLimited(ctx, func() (ethtypes.EthFilterID, error) {
		return gw.target.EthNewBlockFilter(ctx)
	})
}

func (gw *Node) EthNewPendingTransactionFilter(ctx context.Context) (ethtypes.EthFilterID, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return ethtypes.EthFilterID{}, err
	}

	return addUserFilterLimited(ctx, func() (ethtypes.EthFilterID, error) {
		return gw.target.EthNewPendingTransactionFilter(ctx)
	})
}

func (gw *Node) EthUninstallFilter(ctx context.Context, id ethtypes.EthFilterID) (bool, error) {
	if err := gw.limit(ctx, stateRateLimitTokens); err != nil {
		return false, err
	}

	// check if the filter belongs to this connection
	ft := filterTrackerFromContext(ctx)
	ft.lk.Lock()
	defer ft.lk.Unlock()

	if _, ok := ft.userFilters[id]; !ok {
		return false, nil
	}

	ok, err := gw.target.EthUninstallFilter(ctx, id)
	if err != nil {
		return false, err
	}

	delete(ft.userFilters, id)
	return ok, nil
}

func (gw *Node) EthSubscribe(ctx context.Context, eventType string, params *ethtypes.EthSubscriptionParams) (<-chan ethtypes.EthSubscriptionResponse, error) {
	return nil, xerrors.Errorf("not implemented")
}

func (gw *Node) EthUnsubscribe(ctx context.Context, id ethtypes.EthSubscriptionID) (bool, error) {
	return false, xerrors.Errorf("not implemented")
}

var EthMaxFiltersPerConn = 16 // todo make this configurable

func addUserFilterLimited(ctx context.Context, cb func() (ethtypes.EthFilterID, error)) (ethtypes.EthFilterID, error) {
	ft := filterTrackerFromContext(ctx)
	ft.lk.Lock()
	defer ft.lk.Unlock()

	if len(ft.userFilters) >= EthMaxFiltersPerConn {
		return ethtypes.EthFilterID{}, fmt.Errorf("too many filters")
	}

	id, err := cb()
	if err != nil {
		return id, err
	}

	ft.userFilters[id] = time.Now()

	return id, nil
}

func filterTrackerFromContext(ctx context.Context) *filterTracker {
	return ctx.Value(filterTrackerKey).(*filterTracker)
}

type filterTracker struct {
	lk sync.Mutex

	userFilters map[ethtypes.EthFilterID]time.Time
}

// called per request (ws connection)
func newFilterTracker() *filterTracker {
	return &filterTracker{
		userFilters: make(map[ethtypes.EthFilterID]time.Time),
	}
}
