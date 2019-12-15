package gormq

import (
	"context"
	"log"

	"github.com/opentracing/opentracing-go"

	"github.com/streadway/amqp"
)

type bus struct {
	deliveryChan   <-chan amqp.Delivery
	connCloseEvent chan *amqp.Error
	chanCloseEvent chan *amqp.Error
	outChan        chan<- Message
}

// Consumer represents RabbitMQ consumer
type Consumer struct {
	ctx     context.Context
	cfg     ConsumerConfig
	conn    *amqp.Connection
	channel *amqp.Channel

	bus         bus
	reconnector *reconnector

	closing bool
}

// NewConsumer constructs RabbitMq consumer
func NewConsumer(ctx context.Context, cfg ConsumerConfig, ch chan<- Message) (*Consumer, error) {
	c := &Consumer{
		ctx: ctx,
		cfg: cfg,

		bus: bus{
			connCloseEvent: make(chan *amqp.Error),
			chanCloseEvent: make(chan *amqp.Error),
			outChan:        ch,
		},
	}

	if err := c.connect(); err != nil {
		return nil, err
	}

	c.reconnector = newReconnector(
		cfg.MaxReconnectAttempts,
		cfg.ReconnectStrategy,
		cfg.ReconnectTimeout,
		c.connect,
		c.Close,
	)

	return c, nil
}

// Close closes channel and connection
func (c *Consumer) Close() error {
	c.closing = true

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}
	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Consume reads from channel and provides messages into outCh
func (c *Consumer) Consume() (err error) {
	c.bus.deliveryChan, err = c.channel.Consume(
		c.cfg.Queue.Name,
		c.cfg.Queue.Tag,
		c.cfg.AutoAck,
		c.cfg.Queue.Exclusive,
		c.cfg.Queue.NoLocal,
		c.cfg.Queue.NoWait,
		nil,
	)
	if err != nil {
		return
	}

	go c.listen()

	return
}

func (c *Consumer) connect() (err error) {
	if c.conn, err = amqp.Dial(c.cfg.AmqpURI); err != nil {
		return err
	}
	if c.channel, err = c.conn.Channel(); err != nil {
		return err
	}

	c.conn.NotifyClose(c.bus.connCloseEvent)
	c.channel.NotifyClose(c.bus.connCloseEvent)

	if err := c.connect(); err != nil {
		return err
	}
	if err := c.declareExchange(); err != nil {
		return err
	}
	if err := c.declareQueue(); err != nil {
		return err
	}

	return
}

func (c *Consumer) declareExchange() error {
	return c.channel.ExchangeDeclare(
		c.cfg.Exchange.Name,
		c.cfg.Exchange.Type,
		c.cfg.Exchange.Durable,
		c.cfg.Exchange.AutoDelete,
		c.cfg.Exchange.Internal,
		c.cfg.Exchange.NoWait,
		nil,
	)
}

func (c *Consumer) declareQueue() error {
	_, err := c.channel.QueueDeclare(
		c.cfg.Queue.Name,
		c.cfg.Queue.Durable,
		c.cfg.Queue.AutoDelete,
		c.cfg.Queue.Exclusive,
		c.cfg.Queue.NoWait,
		nil,
	)
	if err != nil {
		return err
	}

	return c.channel.QueueBind(
		c.cfg.Queue.Name,
		c.cfg.Queue.Key,
		c.cfg.Exchange.Name,
		c.cfg.Queue.NoWait,
		nil,
	)
}

func (c *Consumer) listen() error {
	for {
		select {
		case delivery := <-c.bus.deliveryChan:
			if c.bus.outChan != nil {
				c.bus.outChan <- c.prepareMessage(delivery)
			}

		case <-c.bus.chanCloseEvent:
			if c.cfg.Reconnect {
				return c.reconnector.reconnect()
			}

		case <-c.bus.connCloseEvent:
			if c.cfg.Reconnect {
				return c.reconnector.reconnect()
			}

		case <-c.ctx.Done():
			return c.Close()
		}
	}
}

func (c *Consumer) prepareMessage(delivery amqp.Delivery) Message {
	msg := buildMessage(delivery)

	if span, err := c.buildSpan(delivery); err == nil {
		msg.SpanIcluded = true
		msg.Span = span
	} else {
		log.Printf("Error building span: %s", err.Error())
	}

	return msg
}

func (c *Consumer) buildSpan(delivery amqp.Delivery) (opentracing.Span, error) {
	textMap := make(map[string]string)

	for k, v := range delivery.Headers {
		if strVal, ok := v.(string); ok {
			textMap[k] = strVal
		}
	}

	wireCtx, err := opentracing.GlobalTracer().Extract(
		opentracing.TextMap,
		opentracing.TextMapCarrier(textMap),
	)
	if err != nil {
		return nil, err
	}

	span := opentracing.StartSpan(
		c.cfg.Name,
		opentracing.FollowsFrom(wireCtx),
	)

	return span, nil
}
