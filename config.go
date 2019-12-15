package gormq

import "time"

// ConsumerConfig represents RabbitMQ consumer configuration model
type ConsumerConfig struct {
	Name                 string
	AmqpURI              string
	Reconnect            bool
	MaxReconnectAttempts int
	ReconnectStrategy    ReconnectStrategy
	ReconnectTimeout     time.Duration
	Exchange             Exchange
	Queue                Queue
	AutoAck              bool
}

// Exchange represents exchange confiuration model
type Exchange struct {
	Name       string
	Type       string
	Durable    bool
	AutoDelete bool
	NoWait     bool
	Internal   bool
}

// Queue represents queue configuration model
type Queue struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	NoLocal    bool
	Key        string
	Tag        string
}
