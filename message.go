package gormq

import (
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)

// Message represents client message model
type Message struct {
	Body        []byte
	Headers     map[string]interface{}
	ContentType string
	RoutingKey  string
	SpanIcluded bool
	Span        opentracing.Span
	Timestamp   time.Time
}

func buildMessage(delivery amqp.Delivery) Message {
	return Message{
		Body:        delivery.Body,
		Headers:     delivery.Headers,
		ContentType: delivery.ContentType,
		RoutingKey:  delivery.RoutingKey,
		Timestamp:   delivery.Timestamp,
	}
}
