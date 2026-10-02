package kafkabasics

import (
	"context"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestConsumeAndProcess_CrashBeforeCommit_RedeliversToNextConsumer(t *testing.T) {
	requireKafka(t)
	topic := uniqueTopic(t, "redelivery")
	group := uniqueTopic(t, "group")

	adm, closeAdm := newAdminClient(t)
	defer closeAdm()
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()
	if _, err := adm.CreateTopics(setupCtx, 1, 1, nil, topic); err != nil {
		t.Fatalf("CreateTopics: %v", err)
	}

	producerClient, err := kgo.NewClient(kgo.SeedBrokers(requireKafka(t)))
	if err != nil {
		t.Fatalf("kgo.NewClient: %v", err)
	}
	defer producerClient.Close()
	producer := NewProducer(producerClient, topic)
	if err := producer.Produce(setupCtx, "order-crash", []byte("process-me")); err != nil {
		t.Fatalf("Produce: %v", err)
	}

	consumer := NewConsumer([]string{requireKafka(t)}, topic)

	// First "consumer process": it processes the one record, then crashes
	// (here: its context is canceled) BEFORE ConsumeAndProcess gets a
	// chance to commit the offset.
	ctx1, cancel1 := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel1()
	firstCalls := 0
	err = consumer.ConsumeAndProcess(ctx1, group, func(ctx context.Context, r *kgo.Record) error {
		firstCalls++
		if string(r.Value) != "process-me" {
			t.Errorf("unexpected record value %q", r.Value)
		}
		cancel1() // simulate a crash: processing succeeded, but we go down right after
		return nil
	})
	if err == nil {
		t.Fatal("ConsumeAndProcess (first consumer) = nil error, want a context-canceled error (it must not have committed)")
	}
	if firstCalls != 1 {
		t.Fatalf("first consumer: handler called %d times, want 1", firstCalls)
	}

	// Second "consumer process": a brand new member of the SAME group. If
	// ConsumeAndProcess correctly withheld the commit above, this one must
	// receive the exact same record again.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel2()
	secondCalls := 0
	_ = consumer.ConsumeAndProcess(ctx2, group, func(ctx context.Context, r *kgo.Record) error {
		secondCalls++
		if string(r.Value) != "process-me" {
			t.Errorf("redelivered record value = %q, want %q", r.Value, "process-me")
		}
		cancel2()
		return nil
	})
	if secondCalls < 1 {
		t.Fatal("second consumer in the same group never received the uncommitted record (no redelivery observed — at-least-once is broken)")
	}
}

func TestConsumeAndProcess_HandlerError_DoesNotCommit(t *testing.T) {
	requireKafka(t)
	topic := uniqueTopic(t, "handler-error")
	group := uniqueTopic(t, "group")

	adm, closeAdm := newAdminClient(t)
	defer closeAdm()
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()
	if _, err := adm.CreateTopics(setupCtx, 1, 1, nil, topic); err != nil {
		t.Fatalf("CreateTopics: %v", err)
	}

	producerClient, err := kgo.NewClient(kgo.SeedBrokers(requireKafka(t)))
	if err != nil {
		t.Fatalf("kgo.NewClient: %v", err)
	}
	defer producerClient.Close()
	producer := NewProducer(producerClient, topic)
	if err := producer.Produce(setupCtx, "order-fail", []byte("boom")); err != nil {
		t.Fatalf("Produce: %v", err)
	}

	consumer := NewConsumer([]string{requireKafka(t)}, topic)
	sentinel := errContextForTest{}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	calls := 0
	err = consumer.ConsumeAndProcess(ctx, group, func(ctx context.Context, r *kgo.Record) error {
		calls++
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("ConsumeAndProcess() = %v, want the handler's own error", err)
	}
	if calls != 1 {
		t.Fatalf("handler called %d times, want 1", calls)
	}

	// A fresh member of the same group must be redelivered the record a
	// failed handler never got to finish.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel2()
	redelivered := false
	_ = consumer.ConsumeAndProcess(ctx2, group, func(ctx context.Context, r *kgo.Record) error {
		redelivered = true
		cancel2()
		return nil
	})
	if !redelivered {
		t.Fatal("record was never redelivered after the handler failed")
	}
}

// errContextForTest is a trivial sentinel error, distinct from context
// errors, used to check that ConsumeAndProcess returns the handler's OWN
// error rather than wrapping or discarding it.
type errContextForTest struct{}

func (errContextForTest) Error() string { return "simulated handler failure" }
