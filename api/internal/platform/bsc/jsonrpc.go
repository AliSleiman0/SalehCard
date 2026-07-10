package bsc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AliSleiman0/salehcard/api/internal/platform/tron"
)

const (
	// transferTopic is keccak256("Transfer(address,address,uint256)") — the
	// ERC20/BEP20 Transfer event signature every token emits.
	transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	jsonrpcTimeout = 10 * time.Second

	// defaultChunkSize keeps each eth_getLogs range safely under the ~5k-block
	// caps common on public BSC nodes.
	defaultChunkSize = 4_000
	// defaultMaxChunksPerTick bounds one ListTransfers call: 16k blocks/tick
	// means a worst-case 7-day backfill (~800k BSC blocks) catches up in ~50
	// watcher ticks (~20 minutes) at a handful of RPC calls per tick.
	defaultMaxChunksPerTick = 4
	// defaultOverlapBlocks re-scans the tail of the previous window every tick
	// so a transfer whose downstream claim/record transiently failed is
	// re-delivered for ~30 ticks. (Downstream dedup is persistent and
	// idempotent — re-delivery is always safe.)
	defaultOverlapBlocks = 1_000

	// blockTimeCacheMax bounds the block→timestamp cache; the overlap re-reads
	// the same few blocks each tick, so hits dominate long before this fills.
	blockTimeCacheMax = 512

	// maxLookbackBlocks bounds how far back the since→block search may reach.
	// The watcher never asks about more than the 7-day late grace (~806k
	// blocks at BSC's ~0.75s block time), and free public nodes prune ancient
	// headers (48Club returns null somewhere between latest−800k and
	// latest−2M), so probing older blocks is both pointless and breaks.
	maxLookbackBlocks = 1_000_000
)

// errBlockUnavailable marks a block the node no longer serves (pruned
// history) — during the since→block search that simply means "older than
// anything we care about".
var errBlockUnavailable = errors.New("bsc: block unavailable (pruned history)")

// defaultRPCEndpoints are free public BSC mainnet JSON-RPC nodes (no API key)
// VERIFIED (2026-07-10) to serve filtered eth_getLogs over ≥4,000-block
// ranges. NOTE: the canonical bsc-dataseed.* nodes are deliberately absent —
// they reject eth_getLogs outright ("limit exceeded" even for 20 blocks), as
// do most free aggregators (publicnode/ankr/zan need keys; 1rpc/blastapi cap
// at 10-50 blocks). 48Club allows 5,000; NodeReal's documented public
// endpoint matches. Overridable via BSC_RPC_ENDPOINTS.
var defaultRPCEndpoints = []string{
	"https://rpc-bsc.48.club",
	"https://bsc-mainnet.nodereal.io/v1/64a9df0874fb4a93b9d0a3849de012d3",
}

// JSONRPCConfig configures the public-node adapter. ChunkSize,
// MaxChunksPerTick and OverlapBlocks are tuning/test knobs (defaulted in
// newJSONRPC, not env-exposed).
type JSONRPCConfig struct {
	Endpoints        []string // default defaultRPCEndpoints
	Contract         string   // token contract to match (default USDTContractBSC)
	MinConfirmations int64    // default 15; also the reorg guard
	ChunkSize        int64
	MaxChunksPerTick int64
	OverlapBlocks    int64
}

// addrCursor is the per-address scan position. initSince remembers which
// `since` the cursor was resolved from so a caller asking about an EARLIER
// window (cannot happen with the current watcher, but cheap to honor) resets
// the cursor.
type addrCursor struct {
	initSince   int64 // unix seconds
	lastScanned int64 // last block fully scanned
}

// jsonrpc reads confirmed BEP20 USDT transfers straight from public BSC
// nodes via eth_getLogs. Unlike the trongrid/etherscan adapters it cannot
// re-read the whole since-window every tick (public getLogs range caps), so
// it keeps an in-memory per-address cursor and walks forward in bounded
// chunks with an overlap. A restart resets the cursor → one bounded re-scan
// of the window; downstream (network,txHash) dedup makes that harmless.
type jsonrpc struct {
	cfg   JSONRPCConfig
	http  *http.Client
	reqID atomic.Int64

	mu          sync.Mutex
	endpointIdx int // last-good endpoint; rotated on failure
	cursors     map[string]*addrCursor
	cachedSince int64 // one-entry since→block cache (mirrors etherscan)
	cachedBlock int64
	blockTimes  map[int64]time.Time
}

func newJSONRPC(cfg JSONRPCConfig) (*jsonrpc, error) {
	if len(cfg.Endpoints) == 0 {
		cfg.Endpoints = defaultRPCEndpoints
	}
	if cfg.Contract == "" {
		cfg.Contract = USDTContractBSC
	}
	if cfg.MinConfirmations <= 0 {
		cfg.MinConfirmations = defaultMinConfirmations
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = defaultChunkSize
	}
	if cfg.MaxChunksPerTick <= 0 {
		cfg.MaxChunksPerTick = defaultMaxChunksPerTick
	}
	if cfg.OverlapBlocks <= 0 {
		cfg.OverlapBlocks = defaultOverlapBlocks
	}
	return &jsonrpc{
		cfg:        cfg,
		http:       &http.Client{Timeout: jsonrpcTimeout},
		cursors:    map[string]*addrCursor{},
		blockTimes: map[int64]time.Time{},
	}, nil
}

// ListTransfers returns confirmed USDT transfers into address discovered in
// this tick's scan window, oldest first (hints are a stub-only concept,
// ignored here). Each on-chain transfer is delivered at least once — recent
// ones repeatedly (overlap) — and never before it has MinConfirmations.
func (j *jsonrpc) ListTransfers(ctx context.Context, address string, since time.Time, _ []tron.AmountHint) ([]tron.Payment, error) {
	latest, err := j.latestBlock(ctx)
	if err != nil {
		return nil, err
	}
	safeHead := latest - j.cfg.MinConfirmations
	if safeHead < 1 {
		return nil, nil
	}

	from, err := j.windowStart(ctx, address, since, safeHead)
	if err != nil {
		return nil, err
	}
	if from > safeHead {
		return nil, nil // nothing new and confirmed yet
	}

	capEnd := from + j.cfg.ChunkSize*j.cfg.MaxChunksPerTick - 1
	to := min(capEnd, safeHead)
	if capEnd < safeHead {
		slog.Warn("bsc: scan window truncated by the per-tick chunk cap — catching up",
			"address", address, "from", from, "to", to, "safeHead", safeHead)
	}

	sinceUnix := since.Unix()
	var payments []tron.Payment
	scanned := from - 1
	for start := from; start <= to; start += j.cfg.ChunkSize {
		end := min(start+j.cfg.ChunkSize-1, to)
		logs, err := j.getLogs(ctx, address, start, end)
		if err != nil {
			// Advance only through fully-scanned chunks; the rest retries
			// next tick from the same position.
			j.advanceCursor(address, scanned, from)
			return payments, err
		}
		for _, lg := range logs {
			p, ok := j.decodeLog(ctx, address, lg)
			if !ok {
				continue
			}
			// The overlap (and closest-before block resolution) reach earlier
			// than `since` — pre-window transfers must not be emitted (parity
			// with trongrid/etherscan, which filter server/client-side).
			if p.BlockTime.Unix() < sinceUnix {
				continue
			}
			payments = append(payments, p)
		}
		scanned = end
	}
	j.advanceCursor(address, scanned, from)
	return payments, nil
}

// windowStart resolves where this tick's scan begins: the block for `since`
// on the first call (or when asked about an earlier window), else the cursor
// minus the overlap.
func (j *jsonrpc) windowStart(ctx context.Context, address string, since time.Time, safeHead int64) (int64, error) {
	sinceUnix := since.Unix()

	j.mu.Lock()
	cur, ok := j.cursors[address]
	if ok && sinceUnix >= cur.initSince {
		from := cur.lastScanned - j.cfg.OverlapBlocks + 1
		j.mu.Unlock()
		if from < 1 {
			from = 1
		}
		return from, nil
	}
	j.mu.Unlock()

	sinceBlock, err := j.resolveSinceBlock(ctx, sinceUnix, safeHead)
	if err != nil {
		return 0, err
	}
	j.mu.Lock()
	j.cursors[address] = &addrCursor{initSince: sinceUnix, lastScanned: sinceBlock - 1}
	j.mu.Unlock()
	return sinceBlock, nil
}

// advanceCursor moves the cursor through the last fully-scanned block. A tick
// that failed before scanning anything (scanned < from) leaves it untouched.
func (j *jsonrpc) advanceCursor(address string, scanned, from int64) {
	if scanned < from {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if cur, ok := j.cursors[address]; ok && scanned > cur.lastScanned {
		cur.lastScanned = scanned
	}
}

// resolveSinceBlock binary-searches the first block whose timestamp is at or
// after sinceUnix (cached per since value — mirrors etherscan's one-entry
// startBlock cache). ~30 light getBlockByNumber calls, paid once per distinct
// since. Estimating at a fixed seconds-per-block is deliberately avoided: BSC
// halved its block time twice in 2025 (Lorentz, Maxwell), so no constant is
// right across those boundaries.
func (j *jsonrpc) resolveSinceBlock(ctx context.Context, sinceUnix, safeHead int64) (int64, error) {
	j.mu.Lock()
	if j.cachedBlock != 0 && j.cachedSince == sinceUnix {
		b := j.cachedBlock
		j.mu.Unlock()
		return b, nil
	}
	j.mu.Unlock()

	lo, hi := max(int64(1), safeHead-maxLookbackBlocks), safeHead
	for lo < hi {
		mid := lo + (hi-lo)/2
		ts, err := j.blockTime(ctx, mid)
		if errors.Is(err, errBlockUnavailable) {
			// Pruned on this node → strictly older than any open intent;
			// treat like a too-old timestamp so the search converges onto
			// the oldest block the node still serves.
			lo = mid + 1
			continue
		}
		if err != nil {
			return 0, err
		}
		if ts.Unix() < sinceUnix {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	j.mu.Lock()
	j.cachedSince, j.cachedBlock = sinceUnix, lo
	j.mu.Unlock()
	return lo, nil
}

// decodeLog turns one eth_getLogs row into a Payment, defensively re-checking
// everything the node already filtered (parity with the other adapters).
func (j *jsonrpc) decodeLog(ctx context.Context, address string, lg logEntry) (tron.Payment, bool) {
	if lg.Removed || len(lg.Topics) < 3 {
		return tron.Payment{}, false
	}
	if !strings.EqualFold(lg.Topics[0], transferTopic) ||
		!strings.EqualFold(lg.Topics[2], padTopicAddress(address)) ||
		!strings.EqualFold(lg.Address, j.cfg.Contract) {
		return tron.Payment{}, false
	}
	micros, ok := hexWeiToMicros(lg.Data)
	if !ok || micros <= 0 {
		return tron.Payment{}, false
	}
	blockNum, err := hexToInt64(lg.BlockNumber)
	if err != nil {
		return tron.Payment{}, false
	}
	blockTime, err := j.blockTime(ctx, blockNum)
	if err != nil {
		slog.Warn("bsc: block timestamp lookup failed — transfer deferred to next tick",
			"block", blockNum, "tx", lg.TransactionHash, "error", err)
		return tron.Payment{}, false
	}
	from := ""
	if len(lg.Topics[1]) == 66 {
		from = "0x" + lg.Topics[1][26:]
	}
	return tron.Payment{
		TxHash:       lg.TransactionHash,
		From:         from,
		AmountMicros: micros,
		BlockTime:    blockTime,
	}, true
}

// blockTime returns a block's timestamp, cached.
func (j *jsonrpc) blockTime(ctx context.Context, n int64) (time.Time, error) {
	j.mu.Lock()
	if ts, ok := j.blockTimes[n]; ok {
		j.mu.Unlock()
		return ts, nil
	}
	j.mu.Unlock()

	var blk *struct {
		Timestamp string `json:"timestamp"`
	}
	if err := j.rpcCall(ctx, "eth_getBlockByNumber", []any{hexBlock(n), false}, &blk); err != nil {
		return time.Time{}, err
	}
	// A null result (or missing timestamp) is how nodes report a block their
	// pruned history no longer covers — not a transport failure.
	if blk == nil || blk.Timestamp == "" {
		return time.Time{}, fmt.Errorf("block %d: %w", n, errBlockUnavailable)
	}
	unix, err := hexToInt64(blk.Timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("bsc: block %d timestamp %q: %w", n, blk.Timestamp, err)
	}
	ts := time.Unix(unix, 0).UTC()

	j.mu.Lock()
	if len(j.blockTimes) >= blockTimeCacheMax {
		j.blockTimes = map[int64]time.Time{} // cheap reset beats an LRU here
	}
	j.blockTimes[n] = ts
	j.mu.Unlock()
	return ts, nil
}

func (j *jsonrpc) latestBlock(ctx context.Context) (int64, error) {
	var hex string
	if err := j.rpcCall(ctx, "eth_blockNumber", []any{}, &hex); err != nil {
		return 0, err
	}
	n, err := hexToInt64(hex)
	if err != nil {
		return 0, fmt.Errorf("bsc: eth_blockNumber returned %q: %w", hex, err)
	}
	return n, nil
}

// logEntry mirrors one eth_getLogs result row.
type logEntry struct {
	Address         string   `json:"address"`
	Topics          []string `json:"topics"`
	Data            string   `json:"data"`
	BlockNumber     string   `json:"blockNumber"`
	TransactionHash string   `json:"transactionHash"`
	Removed         bool     `json:"removed"`
}

// getLogs fetches Transfer events into address for one block range.
func (j *jsonrpc) getLogs(ctx context.Context, address string, from, to int64) ([]logEntry, error) {
	filter := map[string]any{
		"address":   j.cfg.Contract,
		"topics":    []any{transferTopic, nil, padTopicAddress(address)},
		"fromBlock": hexBlock(from),
		"toBlock":   hexBlock(to),
	}
	var logs []logEntry
	if err := j.rpcCall(ctx, "eth_getLogs", []any{filter}, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// rpcError is the JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcCall performs one JSON-RPC request against the current endpoint,
// rotating to the next on any failure (one pass over all endpoints max). The
// last-good endpoint sticks for subsequent calls.
func (j *jsonrpc) rpcCall(ctx context.Context, method string, params []any, result any) error {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      j.reqID.Add(1),
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return fmt.Errorf("bsc: marshal %s: %w", method, err)
	}

	j.mu.Lock()
	start := j.endpointIdx
	j.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt < len(j.cfg.Endpoints); attempt++ {
		idx := (start + attempt) % len(j.cfg.Endpoints)
		if err := j.rpcCallEndpoint(ctx, j.cfg.Endpoints[idx], body, result); err != nil {
			lastErr = fmt.Errorf("bsc: %s via %s: %w", method, j.cfg.Endpoints[idx], err)
			if ctx.Err() != nil {
				return lastErr // don't burn endpoints on a dead context
			}
			continue
		}
		j.mu.Lock()
		j.endpointIdx = idx
		j.mu.Unlock()
		return nil
	}
	return lastErr
}

func (j *jsonrpc) rpcCallEndpoint(ctx context.Context, endpoint string, body []byte, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(raw, 200))
	}

	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if out.Error != nil {
		return fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	if err := json.Unmarshal(out.Result, result); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	return nil
}

// padTopicAddress left-pads an EVM address to the 32-byte topic shape.
func padTopicAddress(addr string) string {
	return "0x000000000000000000000000" + strings.ToLower(strings.TrimPrefix(addr, "0x"))
}

func hexBlock(n int64) string { return "0x" + strconv.FormatInt(n, 16) }

func hexToInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(s), "0x"), 16, 64)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
