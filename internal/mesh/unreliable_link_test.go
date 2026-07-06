package mesh

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestUnreliableLinkReconnectSendsOnlyMissingCurrentSummaries(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	sender := newUnreliableLinkNode(t, "boat-beta")
	receiver := newUnreliableLinkNode(t, "boat-alpha")

	alreadyKnown := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	newerTelemetry := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-2", 2, observed.Add(time.Second), map[string]any{"battery": 84})
	missingAlert := mustMeshItem(t, "boat-002", "sys-002", "alert.low-battery", "alert-1", 1, observed.Add(2*time.Second), map[string]any{"severity": "warning"})

	sender.publish(t, alreadyKnown, newerTelemetry, missingAlert)
	receiver.publish(t, alreadyKnown)

	diff := exchangeOverUnreliableLink(t, sender, receiver, DiffOptions{
		Now: observed.Add(3 * time.Second),
	}, 1)
	if got, want := len(diff.Items), 2; got != want {
		t.Fatalf("reconnect diff item count = %d, want %d", got, want)
	}
	if got, fullGraph := len(diff.Items), sender.summaries.Len(); got >= fullGraph {
		t.Fatalf("reconnect sent %d items, want fewer than full current graph %d", got, fullGraph)
	}
	assertHasEntity(t, diff.Items, "vehicle-2")
	assertHasEntity(t, diff.Items, "alert-1")

	again, err := sender.summaries.Diff(receiver.summaries.Watermarks(observed.Add(3*time.Second)), DiffOptions{
		Now: observed.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("Diff() after catch-up error = %v", err)
	}
	if len(again.Items) != 0 {
		t.Fatalf("post-catch-up diff item count = %d, want 0", len(again.Items))
	}
}

func TestUnreliableLinkDuplicateDeliveryIsIdempotent(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	sender := newUnreliableLinkNode(t, "boat-beta")
	receiver := newUnreliableLinkNode(t, "boat-alpha")

	item := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	sender.publish(t, item)

	diff := exchangeOverUnreliableLink(t, sender, receiver, DiffOptions{
		Now: observed.Add(time.Second),
	}, 1)
	afterFirstDelivery := receiver.summaries.Watermarks(observed.Add(time.Second))

	receiver.deliver(t, diff.Items...)
	afterDuplicateDelivery := receiver.summaries.Watermarks(observed.Add(time.Second))

	if !reflect.DeepEqual(afterDuplicateDelivery, afterFirstDelivery) {
		t.Fatalf("duplicate delivery changed watermarks\nfirst: %#v\nduplicate: %#v", afterFirstDelivery, afterDuplicateDelivery)
	}
	if got, want := receiver.summaries.Len(), 1; got != want {
		t.Fatalf("receiver summary count after duplicate = %d, want %d", got, want)
	}
}

func TestUnreliableLinkOmitsExpiredTelemetryOnReconnect(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	sender := newUnreliableLinkNode(t, "boat-beta")
	receiver := newUnreliableLinkNode(t, "boat-alpha")

	expired := mustMeshItemWithTTL(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, 5*time.Second, map[string]any{"battery": 91})
	sender.publish(t, expired)

	reconnectedAt := observed.Add(10 * time.Second)
	diff := exchangeOverUnreliableLink(t, sender, receiver, DiffOptions{
		Now: reconnectedAt,
	}, 1)
	if len(diff.Items) != 0 {
		t.Fatalf("expired reconnect diff item count = %d, want 0", len(diff.Items))
	}
	if got := len(sender.summaries.Watermarks(reconnectedAt).Entries); got != 0 {
		t.Fatalf("sender watermark count after expiry = %d, want 0", got)
	}
	if got := len(receiver.summaries.Watermarks(reconnectedAt).Entries); got != 0 {
		t.Fatalf("receiver watermark count after expiry = %d, want 0", got)
	}
}

func TestUnreliableLinkBoundedCatchUpProgressesAcrossReconnects(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	sender := newUnreliableLinkNode(t, "boat-beta")
	receiver := newUnreliableLinkNode(t, "boat-alpha")

	for seq := uint64(1); seq <= 5; seq++ {
		entityID := "vehicle-" + strconv.FormatUint(seq, 10)
		sender.publish(t, mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", entityID, seq, observed.Add(time.Duration(seq)*time.Second), map[string]any{"seq": seq}))
	}

	first := exchangeOverUnreliableLink(t, sender, receiver, DiffOptions{
		Now:      observed.Add(10 * time.Second),
		MaxItems: 2,
	}, 1)
	if got, want := len(first.Items), 2; got != want || !first.Truncated {
		t.Fatalf("first catch-up len/truncated = %d/%v, want %d/true", got, first.Truncated, want)
	}
	assertCellItemCount(t, receiver, observed.Add(10*time.Second), 2)

	second := exchangeOverUnreliableLink(t, sender, receiver, DiffOptions{
		Now:      observed.Add(10 * time.Second),
		MaxItems: 2,
	}, 1)
	if got, want := len(second.Items), 2; got != want || !second.Truncated {
		t.Fatalf("second catch-up len/truncated = %d/%v, want %d/true", got, second.Truncated, want)
	}
	assertCellItemCount(t, receiver, observed.Add(10*time.Second), 4)

	third := exchangeOverUnreliableLink(t, sender, receiver, DiffOptions{
		Now:      observed.Add(10 * time.Second),
		MaxItems: 2,
	}, 1)
	if got, want := len(third.Items), 1; got != want || third.Truncated {
		t.Fatalf("third catch-up len/truncated = %d/%v, want %d/false", got, third.Truncated, want)
	}
	assertCellItemCount(t, receiver, observed.Add(10*time.Second), 5)

	complete, err := sender.summaries.Diff(receiver.summaries.Watermarks(observed.Add(10*time.Second)), DiffOptions{
		Now:      observed.Add(10 * time.Second),
		MaxItems: 2,
	})
	if err != nil {
		t.Fatalf("Diff() after bounded catch-up error = %v", err)
	}
	if len(complete.Items) != 0 || complete.Truncated {
		t.Fatalf("complete diff len/truncated = %d/%v, want 0/false", len(complete.Items), complete.Truncated)
	}
}

type unreliableLinkNode struct {
	id        string
	summaries *SummaryIndex
}

func newUnreliableLinkNode(t *testing.T, id string) *unreliableLinkNode {
	t.Helper()

	if id == "" {
		t.Fatal("unreliable link node id is required")
	}
	return &unreliableLinkNode{
		id:        id,
		summaries: NewSummaryIndex(),
	}
}

func (n *unreliableLinkNode) publish(t *testing.T, items ...Item) {
	t.Helper()

	n.deliver(t, items...)
}

func (n *unreliableLinkNode) deliver(t *testing.T, items ...Item) {
	t.Helper()

	for _, item := range items {
		if err := n.summaries.Upsert(item); err != nil {
			t.Fatalf("%s Upsert() error = %v", n.id, err)
		}
	}
}

func exchangeOverUnreliableLink(t *testing.T, sender, receiver *unreliableLinkNode, opts DiffOptions, duplicateDeliveries int) Diff {
	t.Helper()

	diff, err := sender.summaries.Diff(receiver.summaries.Watermarks(opts.Now), opts)
	if err != nil {
		t.Fatalf("Diff(%s -> %s) error = %v", sender.id, receiver.id, err)
	}
	for i := 0; i < duplicateDeliveries; i++ {
		receiver.deliver(t, diff.Items...)
	}
	return diff
}

func mustMeshItemWithTTL(t *testing.T, originNode, originVehicle, predicateGroup, entityID string, sequence uint64, observed time.Time, ttl time.Duration, payload any) Item {
	t.Helper()

	item, err := NewItem(Envelope{
		OriginNodeID:    originNode,
		OriginVehicleID: originVehicle,
		EntityID:        entityID,
		PredicateGroup:  predicateGroup,
		OriginSequence:  sequence,
		OperationID:     originNode + ":" + predicateGroup + ":" + entityID,
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(ttl),
		SourceKind:      SourceKindMAVLinkProjection,
		Confidence:      1,
		MergePolicy:     MergePolicyLastWriterWins,
	}, payload)
	if err != nil {
		t.Fatalf("NewItem() error = %v", err)
	}
	return item
}

func assertHasEntity(t *testing.T, items []Item, entityID string) {
	t.Helper()

	for _, item := range items {
		if item.Envelope.EntityID == entityID {
			return
		}
	}
	t.Fatalf("diff missing entity %q", entityID)
}

func assertCellItemCount(t *testing.T, node *unreliableLinkNode, now time.Time, want int) {
	t.Helper()

	watermarks := node.summaries.Watermarks(now)
	if len(watermarks.Entries) != 1 {
		t.Fatalf("watermark entry count = %d, want 1: %#v", len(watermarks.Entries), watermarks)
	}
	if got := watermarks.Entries[0].ItemCount; got != want {
		t.Fatalf("watermark item count = %d, want %d", got, want)
	}
}
