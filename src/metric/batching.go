package metric

import (
	"fmt"
	"time"

	v2 "plumbing/v2"
)

type IncrementOpt func(*incrementOption)

type incrementOption struct {
	delta uint64
}

func WithIncrement(delta uint64) func(*incrementOption) {
	return func(i *incrementOption) {
		i.delta = delta
	}
}

func IncCounter(name string, options ...IncrementOpt) {
	if batchBuffer == nil {
		return
	}

	incConf := &incrementOption{
		delta: 1,
	}

	for _, opt := range options {
		opt(incConf)
	}

	e := &v2.Envelope{
		SourceUuid: conf.sourceUUID,
		Timestamp:  time.Now().UnixNano(),
		Message: &v2.Envelope_Counter{
			Counter: &v2.Counter{
				Name: fmt.Sprintf("%s.%s", conf.prefix, name),
				Value: &v2.Counter_Delta{
					Delta: incConf.delta,
				},
			},
		},
		Tags: map[string]*v2.Value{
			"component": &v2.Value{
				Data: &v2.Value_Text{
					Text: conf.component,
				},
			},
		},
	}

	batchBuffer.Set(e)
}

func runBatcher() {
	ticker := time.NewTicker(conf.batchInterval)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		s := sender
		mu.Unlock()

		if s == nil {
			continue
		}

		for _, e := range aggregateCounters() {
			s.Send(e)
		}
	}
}

func aggregateCounters() map[string]*v2.Envelope {
	m := make(map[string]*v2.Envelope)
	for {
		envelope, ok := batchBuffer.TryNext()
		if !ok {
			break
		}

		existingEnvelope, ok := m[envelope.GetCounter().Name]
		if !ok {
			existingEnvelope = envelope
			m[envelope.GetCounter().Name] = existingEnvelope
			continue
		}

		existingEnvelope.GetCounter().GetValue().(*v2.Counter_Delta).Delta += envelope.GetCounter().GetDelta()
	}

	return m
}
