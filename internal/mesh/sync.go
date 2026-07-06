package mesh

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type CellID struct {
	OriginNodeID    string `json:"origin_node_id"`
	OriginVehicleID string `json:"origin_vehicle_id"`
	PredicateGroup  string `json:"predicate_group"`
}

type CellWatermark struct {
	CellID
	OriginSequence uint64             `json:"origin_sequence,omitempty"`
	HybridTime     *HybridLogicalTime `json:"hybrid_logical_time,omitempty"`
	ObservedAt     time.Time          `json:"observed_at"`
	OperationID    string             `json:"operation_id"`
	ItemCount      int                `json:"item_count"`
	CellHash       string             `json:"cell_hash"`
}

type WatermarkSet struct {
	Entries []CellWatermark `json:"entries"`
}

type DiffOptions struct {
	Now      time.Time
	MaxItems int
}

type Diff struct {
	Items     []Item `json:"items"`
	Truncated bool   `json:"truncated"`
}

type SummaryIndex struct {
	mu    sync.RWMutex
	items map[summaryKey]Item
}

type summaryKey struct {
	CellID
	EntityID string
}

func NewSummaryIndex() *SummaryIndex {
	return &SummaryIndex{items: make(map[summaryKey]Item)}
}

func (e Envelope) CellID() CellID {
	return CellID{
		OriginNodeID:    e.OriginNodeID,
		OriginVehicleID: e.OriginVehicleID,
		PredicateGroup:  e.PredicateGroup,
	}
}

func (i Item) CellID() CellID {
	return i.Envelope.CellID()
}

func (idx *SummaryIndex) Upsert(item Item) error {
	if err := item.Validate(); err != nil {
		return err
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	key := item.summaryKey()
	if existing, ok := idx.items[key]; ok && !itemNewerThanWatermark(item, watermarkFromItem(existing)) {
		return nil
	}
	idx.items[key] = cloneItem(item)
	return nil
}

func (idx *SummaryIndex) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.items)
}

func (idx *SummaryIndex) Watermarks(now time.Time) WatermarkSet {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return watermarksForItems(idx.currentItems(now))
}

func (idx *SummaryIndex) Diff(peer WatermarkSet, opts DiffOptions) (Diff, error) {
	if err := peer.Validate(); err != nil {
		return Diff{}, err
	}

	idx.mu.RLock()
	items := idx.currentItems(opts.Now)
	idx.mu.RUnlock()

	localWatermarks := watermarksForItems(items)
	byCell := make(map[CellID][]Item)
	for _, item := range items {
		cell := item.CellID()
		byCell[cell] = append(byCell[cell], item)
	}

	var selected []Item
	for _, local := range localWatermarks.Entries {
		cellItems := byCell[local.CellID]
		peerWatermark, ok := peer.Get(local.CellID)
		switch {
		case !ok:
			selected = append(selected, cellItems...)
		case local.sameSyncState(peerWatermark):
			continue
		case watermarkNewerThan(local, peerWatermark):
			for _, item := range cellItems {
				if itemNewerThanWatermark(item, peerWatermark) {
					selected = append(selected, item)
				}
			}
		case local.digestDiffers(peerWatermark):
			selected = append(selected, cellItems...)
		}
	}

	sortItems(selected)
	if opts.MaxItems > 0 && len(selected) > opts.MaxItems {
		return Diff{
			Items:     cloneItems(selected[:opts.MaxItems]),
			Truncated: true,
		}, nil
	}
	return Diff{Items: cloneItems(selected)}, nil
}

func (s WatermarkSet) Get(cell CellID) (CellWatermark, bool) {
	for _, entry := range s.Entries {
		if entry.CellID == cell {
			return entry, true
		}
	}
	return CellWatermark{}, false
}

func (s WatermarkSet) Validate() error {
	seen := make(map[CellID]struct{}, len(s.Entries))
	for _, entry := range s.Entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		if _, ok := seen[entry.CellID]; ok {
			return fmt.Errorf("duplicate watermark for origin/state cell %s", entry.CellID.String())
		}
		seen[entry.CellID] = struct{}{}
	}
	return nil
}

func (w CellWatermark) Validate() error {
	if err := w.CellID.Validate(); err != nil {
		return err
	}
	if w.OriginSequence == 0 && (w.HybridTime == nil || w.HybridTime.IsZero()) {
		return errors.New("watermark requires origin sequence or hybrid logical time")
	}
	if w.OperationID == "" {
		return errors.New("watermark operation_id is required")
	}
	if w.ObservedAt.IsZero() {
		return errors.New("watermark observed_at is required")
	}
	if w.ItemCount <= 0 {
		return fmt.Errorf("watermark item_count must be positive, got %d", w.ItemCount)
	}
	if !validSHA256(w.CellHash) {
		return fmt.Errorf("watermark cell_hash must be sha256:<hex>, got %q", w.CellHash)
	}
	return nil
}

func (c CellID) Validate() error {
	var missing []string
	for name, value := range map[string]string{
		"origin_node_id":    c.OriginNodeID,
		"origin_vehicle_id": c.OriginVehicleID,
		"predicate_group":   c.PredicateGroup,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("origin/state cell missing %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c CellID) String() string {
	return c.OriginNodeID + "/" + c.OriginVehicleID + "/" + c.PredicateGroup
}

func (idx *SummaryIndex) currentItems(now time.Time) []Item {
	items := make([]Item, 0, len(idx.items))
	for _, item := range idx.items {
		if now.IsZero() || item.Envelope.ExpiresAt.After(now) {
			items = append(items, cloneItem(item))
		}
	}
	sortItems(items)
	return items
}

func (i Item) summaryKey() summaryKey {
	return summaryKey{
		CellID:   i.CellID(),
		EntityID: i.Envelope.EntityID,
	}
}

func watermarksForItems(items []Item) WatermarkSet {
	byCell := make(map[CellID][]Item)
	for _, item := range items {
		cell := item.CellID()
		byCell[cell] = append(byCell[cell], item)
	}

	watermarks := WatermarkSet{Entries: make([]CellWatermark, 0, len(byCell))}
	for cell, cellItems := range byCell {
		sortItems(cellItems)
		watermark := watermarkFromItem(cellItems[0])
		for _, item := range cellItems[1:] {
			if itemNewerThanWatermark(item, watermark) {
				watermark = watermarkFromItem(item)
			}
		}
		watermark.CellID = cell
		watermark.ItemCount = len(cellItems)
		watermark.CellHash = hashCell(cellItems)
		watermarks.Entries = append(watermarks.Entries, watermark)
	}

	sort.Slice(watermarks.Entries, func(i, j int) bool {
		return watermarks.Entries[i].CellID.String() < watermarks.Entries[j].CellID.String()
	})
	return watermarks
}

func watermarkFromItem(item Item) CellWatermark {
	return CellWatermark{
		CellID:         item.CellID(),
		OriginSequence: item.Envelope.OriginSequence,
		HybridTime:     cloneHybridTime(item.Envelope.HybridTime),
		ObservedAt:     item.Envelope.ObservedAt,
		OperationID:    item.Envelope.OperationID,
		ItemCount:      1,
		CellHash:       hashCell([]Item{item}),
	}
}

func itemNewerThanWatermark(item Item, watermark CellWatermark) bool {
	return watermarkNewerThan(watermarkFromItem(item), watermark)
}

func watermarkNewerThan(left, right CellWatermark) bool {
	if left.OriginSequence > 0 || right.OriginSequence > 0 {
		if left.OriginSequence != right.OriginSequence {
			return left.OriginSequence > right.OriginSequence
		}
		return left.OperationID > right.OperationID
	}

	if left.HybridTime != nil || right.HybridTime != nil {
		return compareHybridTime(left.HybridTime, right.HybridTime) > 0
	}

	if !left.ObservedAt.Equal(right.ObservedAt) {
		return left.ObservedAt.After(right.ObservedAt)
	}
	return left.OperationID > right.OperationID
}

func compareHybridTime(left, right *HybridLogicalTime) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return -1
	case right == nil:
		return 1
	}
	if !left.WallTime.Equal(right.WallTime) {
		if left.WallTime.Before(right.WallTime) {
			return -1
		}
		return 1
	}
	if left.Counter < right.Counter {
		return -1
	}
	if left.Counter > right.Counter {
		return 1
	}
	return 0
}

func (w CellWatermark) sameSyncState(peer CellWatermark) bool {
	return w.OriginSequence == peer.OriginSequence &&
		compareHybridTime(w.HybridTime, peer.HybridTime) == 0 &&
		w.OperationID == peer.OperationID &&
		w.ItemCount == peer.ItemCount &&
		w.CellHash == peer.CellHash
}

func (w CellWatermark) digestDiffers(peer CellWatermark) bool {
	return w.ItemCount != peer.ItemCount || w.CellHash != peer.CellHash
}

func hashCell(items []Item) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{
			item.Envelope.EntityID,
			item.Envelope.PredicateGroup,
			item.Envelope.OperationID,
			item.Envelope.PayloadHash,
		}, "\x00"))
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i].Envelope
		right := items[j].Envelope
		if left.OriginNodeID != right.OriginNodeID {
			return left.OriginNodeID < right.OriginNodeID
		}
		if left.OriginVehicleID != right.OriginVehicleID {
			return left.OriginVehicleID < right.OriginVehicleID
		}
		if left.PredicateGroup != right.PredicateGroup {
			return left.PredicateGroup < right.PredicateGroup
		}
		if left.EntityID != right.EntityID {
			return left.EntityID < right.EntityID
		}
		if left.OriginSequence != right.OriginSequence {
			return left.OriginSequence < right.OriginSequence
		}
		if cmp := compareHybridTime(left.HybridTime, right.HybridTime); cmp != 0 {
			return cmp < 0
		}
		return left.OperationID < right.OperationID
	})
}

func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	for i, item := range items {
		out[i] = cloneItem(item)
	}
	return out
}

func cloneItem(item Item) Item {
	out := item
	out.Payload = append(item.Payload[:0:0], item.Payload...)
	out.Envelope.HybridTime = cloneHybridTime(item.Envelope.HybridTime)
	if item.Envelope.SemStreams != nil {
		semstreams := *item.Envelope.SemStreams
		out.Envelope.SemStreams = &semstreams
	}
	return out
}

func cloneHybridTime(in *HybridLogicalTime) *HybridLogicalTime {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
