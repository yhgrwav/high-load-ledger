package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// KafkaProducer is used by gateway after a successful transfer.
type KafkaProducer struct {
	Brokers string `env:"KAFKA_BROKERS" envDefault:"kafka:29092"`
	Topic   string `env:"KAFKA_TOPIC" envDefault:"completed_transactions"`
}

func LoadKafkaProducer() (KafkaProducer, error) {
	var cfg KafkaProducer
	if err := env.Parse(&cfg); err != nil {
		return KafkaProducer{}, fmt.Errorf("kafka producer config: %w", err)
	}
	return cfg, nil
}

func (k KafkaProducer) BrokerList() []string {
	return splitAddrs(k.Brokers)
}

type KafkaConsumer struct {
	Brokers string `env:"KAFKA_BROKERS" envDefault:"kafka:29092"`
	Topic   string `env:"KAFKA_TOPIC" envDefault:"completed_transactions"`
	GroupID string `env:"KAFKA_CONSUMER_GROUP" envDefault:"ledger-stats-cache"`
}

func LoadKafkaConsumer() (KafkaConsumer, error) {
	var cfg KafkaConsumer
	if err := env.Parse(&cfg); err != nil {
		return KafkaConsumer{}, fmt.Errorf("kafka consumer config: %w", err)
	}
	return cfg, nil
}

func (k KafkaConsumer) BrokerList() []string {
	return splitAddrs(k.Brokers)
}

func splitAddrs(addrs string) []string {
	return strings.Split(addrs, ",")
}
