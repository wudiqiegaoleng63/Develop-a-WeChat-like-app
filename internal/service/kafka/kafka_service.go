package kafka

import (
	"context"
	myconfig "gochat/internal/config"
	"gochat/pkg/zlog"
	"time"

	"github.com/segmentio/kafka-go"
)

var ctx = context.Background()

type kafkaService struct {
	ChatWriter 	*kafka.Writer
	ChatReader 	*kafka.Reader
	KafkaConn	*kafka.Conn
}

var KafkaService = new(kafkaService)

func (k *kafkaService) KafkaInit() {
	kafkaConfig := myconfig.GetConfig().KafkaConfig

	// 创建生产者 Writer
	k.ChatWriter = &kafka.Writer{
		Addr: kafka.TCP(kafkaConfig.HostPort),
		Topic: kafkaConfig.ChatTopic,
		Balancer: &kafka.Hash{},
		WriteTimeout: kafkaConfig.Timeout * time.Second,
		RequiredAcks: kafka.RequireNone,
		AllowAutoTopicCreation: false,
	}


	// 创建消费者
	k.ChatReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaConfig.HostPort},
		Topic:  kafkaConfig.ChatTopic,
		CommitInterval: kafkaConfig.Timeout * time.Second,
		GroupID: "chat",
		StartOffset: kafka.LastOffset,
	})



}

// 关闭kafka服务
func (k *kafkaService) KafkaClose() {
	if err := k.ChatWriter.Close(); err != nil {
		zlog.Error(err.Error())
	}

	if err := k.ChatReader.Close(); err != nil {
		zlog.Error(err.Error())
	}
}


// CreateTopic
func (k *kafkaService) CreateTopic() {
	kafkaConfig := myconfig.GetConfig().KafkaConfig
	chatTopic := kafkaConfig.ChatTopic

	// 连接任意kafka节点
	var err error
	k.KafkaConn, err = kafka.Dial("tcp", kafkaConfig.HostPort)

	if err != nil {
		zlog.Error(err.Error())
	}

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:	chatTopic,
			NumPartitions: kafkaConfig.Partition,
			ReplicationFactor: 1,
		},
	}

	if err = k.KafkaConn.CreateTopics(topicConfigs...); err != nil {
		zlog.Error(err.Error())
	}
}

