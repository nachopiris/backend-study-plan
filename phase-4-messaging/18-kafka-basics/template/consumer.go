package kafkabasics

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Handler processes one consumed record. If it returns a non-nil error,
// ConsumeAndProcess must treat that record (and the delivery it is part of)
// as NOT successfully processed: its offset — and the offset of every record
// after it in the same poll batch — must not be committed.
type Handler func(ctx context.Context, record *kgo.Record) error

// Consumer holds the Kafka connection details needed to consume one topic.
// Each call to ConsumeAndProcess joins group as a brand-new consumer group
// member (a fresh underlying client) — exactly the way a freshly started
// process would — which is what makes it possible to simulate "a consumer
// crashed and a new one took over" in a test: call ConsumeAndProcess once,
// stop it, then call it again with the same group.
type Consumer struct {
	SeedBrokers []string
	Topic       string
}

// NewConsumer builds a Consumer for topic over seedBrokers.
func NewConsumer(seedBrokers []string, topic string) *Consumer {
	return &Consumer{SeedBrokers: seedBrokers, Topic: topic}
}

// ConsumeAndProcess joins group as a new member consuming c.Topic, with
// autocommit DISABLED, and calls handler for every record it receives, in
// the order franz-go delivers them.
//
// Contract:
//   - For every record in a polled batch, in order: call handler. If handler
//     returns an error, ConsumeAndProcess stops immediately and returns that
//     error, WITHOUT committing the offset of the failing record, nor of any
//     later record (even one from a different partition) in that same
//     batch. Nothing from a previous, already-committed batch is affected.
//   - After handler has returned nil for every record in a polled batch,
//     ConsumeAndProcess checks ctx: if ctx is already done at that point, it
//     returns ctx.Err() WITHOUT committing that batch's offsets. This models
//     a crash between "finished the work" and "recorded that fact": the next
//     consumer to join group will be redelivered every record in that batch,
//     exactly the at-least-once behavior this practice asks you to observe.
//   - Otherwise (ctx still live after a successful batch), commit that
//     batch's offsets (so a future consumer resumes AFTER it) before polling
//     again.
//   - A poll that returns no records and no error is not a failure: keep
//     polling until ctx is done.
//   - ConsumeAndProcess returns nil only when ctx is canceled strictly
//     BETWEEN polls, with nothing pending to commit.
//
// Not implemented yet: returns an error immediately without connecting to
// Kafka at all.
func (c *Consumer) ConsumeAndProcess(ctx context.Context, group string, handler Handler) error {
	return errors.New("not implemented")
}
