// Package kafkabasics implements the two halves of a minimal, direct
// franz-go producer/consumer pair over a real single-node Kafka broker
// (KRaft mode, no Zookeeper): a keyed producer and a manual-offset-commit
// consumer. See the README for how partitions, consumer groups, and offsets
// fit together.
package kafkabasics

import (
	"context"
	"errors"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer publishes order-related messages to one fixed Kafka topic.
type Producer struct {
	client *kgo.Client
	topic  string
}

// NewProducer builds a Producer that publishes to topic using client. client
// is not owned by Producer: the caller is responsible for closing it.
func NewProducer(client *kgo.Client, topic string) *Producer {
	return &Producer{client: client, topic: topic}
}

// Produce publishes payload for orderID to the topic, synchronously (it does
// not return until the broker has acknowledged the write or an error
// occurs).
//
// Contract:
//   - The record's key MUST be orderID (as bytes), unmodified. Kafka's
//     default partitioner hashes the key to choose a partition, so every
//     record sharing the same orderID lands on the SAME partition — which is
//     what makes per-order ordering possible at all (Kafka only guarantees
//     order WITHIN a partition, never across the whole topic).
//   - The record's value MUST be payload, unmodified.
//   - Produce returns a non-nil error if the broker fails to acknowledge the
//     write (including if ctx is canceled first).
//
// Not implemented yet: always returns an error without producing anything.
func (p *Producer) Produce(ctx context.Context, orderID string, payload []byte) error {
	return errors.New("not implemented")
}
