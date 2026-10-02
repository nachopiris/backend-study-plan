package kafkabasics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// newAdminClient returns a kadm client for topic management, backed by a
// fresh kgo.Client that the caller must close via the returned cleanup func.
func newAdminClient(t *testing.T) (*kadm.Client, func()) {
	t.Helper()
	cl, err := kgo.NewClient(kgo.SeedBrokers(requireKafka(t)))
	if err != nil {
		t.Fatalf("kgo.NewClient: %v", err)
	}
	return kadm.NewClient(cl), cl.Close
}

func TestProducer_SameKey_SamePartition_InOrder(t *testing.T) {
	requireKafka(t)
	topic := uniqueTopic(t, "orders")

	adm, closeAdm := newAdminClient(t)
	defer closeAdm()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := adm.CreateTopics(ctx, 3, 1, nil, topic); err != nil {
		t.Fatalf("CreateTopics: %v", err)
	}

	producerClient, err := kgo.NewClient(kgo.SeedBrokers(requireKafka(t)))
	if err != nil {
		t.Fatalf("kgo.NewClient: %v", err)
	}
	defer producerClient.Close()
	producer := NewProducer(producerClient, topic)

	// Two different orders, interleaved, five messages each: if Produce
	// keys correctly, every message for order-a lands in the SAME
	// partition as every other message for order-a (same for order-b),
	// in the order they were produced.
	const perOrder = 5
	for i := 0; i < perOrder; i++ {
		if err := producer.Produce(ctx, "order-a", []byte(fmt.Sprintf("a-seq-%d", i))); err != nil {
			t.Fatalf("Produce(order-a, %d): %v", i, err)
		}
		if err := producer.Produce(ctx, "order-b", []byte(fmt.Sprintf("b-seq-%d", i))); err != nil {
			t.Fatalf("Produce(order-b, %d): %v", i, err)
		}
	}

	// Consume everything back (no group: a plain partition consumer
	// reading from the start) and check per-order partition and order.
	consumerClient, err := kgo.NewClient(
		kgo.SeedBrokers(requireKafka(t)),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		t.Fatalf("kgo.NewClient (consumer): %v", err)
	}
	defer consumerClient.Close()

	type seen struct {
		partition int32
		seqs      []string
	}
	byOrder := map[string]*seen{}
	total := 0
	deadline := time.Now().Add(20 * time.Second)
	for total < 2*perOrder && time.Now().Before(deadline) {
		pollCtx, pollCancel := context.WithTimeout(ctx, 5*time.Second)
		fetches := consumerClient.PollFetches(pollCtx)
		pollCancel()
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				t.Logf("fetch error (continuing): %v", e)
			}
		}
		fetches.EachRecord(func(r *kgo.Record) {
			order := string(r.Key)
			s, ok := byOrder[order]
			if !ok {
				s = &seen{partition: r.Partition}
				byOrder[order] = s
			}
			if s.partition != r.Partition {
				t.Errorf("order %q: record landed on partition %d, but an earlier record for the same key landed on partition %d (same key must always map to the same partition)", order, r.Partition, s.partition)
			}
			s.seqs = append(s.seqs, string(r.Value))
			total++
		})
	}

	for _, order := range []string{"order-a", "order-b"} {
		s, ok := byOrder[order]
		if !ok {
			t.Fatalf("never received any record for %q", order)
		}
		if len(s.seqs) != perOrder {
			t.Fatalf("%q: received %d records, want %d: %v", order, len(s.seqs), perOrder, s.seqs)
		}
		for i, v := range s.seqs {
			want := fmt.Sprintf("%s-seq-%d", order[len(order)-1:], i)
			if v != want {
				t.Errorf("%q: record %d = %q, want %q (records for one key must stay in the order they were produced)", order, i, v, want)
			}
		}
	}
}
