// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSBus adapts a NATS connection to the Publisher seam and exposes the
// synchronous subscription the worker's run loop pulls from. It is the ONLY file
// in this package that imports the NATS client: the orchestration in worker.go
// stays transport-agnostic, so the job lifecycle is unit-tested against a fake
// Publisher and the real roundtrip is covered by the smoke-test.
type NATSBus struct {
	conn *nats.Conn
}

// ConnectNATS dials url and returns a NATSBus. name labels the connection for
// server-side observability; drainTimeout bounds the graceful drain on shutdown.
// The worker keeps reconnecting indefinitely — the motor owns restart policy, so
// a transient NATS blip should not crash the container.
func ConnectNATS(url, name string, drainTimeout time.Duration) (*NATSBus, error) {
	conn, err := nats.Connect(url,
		nats.Name(name),
		nats.MaxReconnects(-1),
		nats.RetryOnFailedConnect(true),
		nats.DrainTimeout(drainTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("worker: connect nats %q: %w", url, err)
	}
	return &NATSBus{conn: conn}, nil
}

// Publish satisfies Publisher: it sends one envelope to a subject.
func (b *NATSBus) Publish(subject string, data []byte) error {
	return b.conn.Publish(subject, data)
}

// SubscribeSync subscribes synchronously to subject; the caller pulls messages
// with (*nats.Subscription).NextMsgWithContext, one at a time — the single-job
// contract.
func (b *NATSBus) SubscribeSync(subject string) (*nats.Subscription, error) {
	sub, err := b.conn.SubscribeSync(subject)
	if err != nil {
		return nil, fmt.Errorf("worker: subscribe %q: %w", subject, err)
	}
	return sub, nil
}

// Drain flushes in-flight publishes and closes the connection within the
// configured drain timeout. Safe to defer.
func (b *NATSBus) Drain() error {
	return b.conn.Drain()
}
