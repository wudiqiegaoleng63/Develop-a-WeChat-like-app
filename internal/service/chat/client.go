package chat

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"kama-chat-server/internal/config"
	"kama-chat-server/internal/dto/request"
	myKafka "kama-chat-server/internal/service/kafka"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/zlog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)


type Client struct {
	Conn 	*websocket.Conn
	Uuid	string
	SendTo	chan	[]byte
	SendBack chan	*MessageBack
}

type MessageBack struct {
	Message 	[]byte
	Uuid		string
}

var messageMode = config.GetConfig().KafkaConfig.MessageMode

var ctx = context.Background()

var upgrader = websocket.Upgrader{
	ReadBufferSize: 2048,
	WriteBufferSize: 2048,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


// 读取websocket 消息， 根据mode选择写入kafka还是channel
func (c *Client) Read() {
	zlog.Info("ws read goroutine start")
	for {
		_, jsonMessage, err := c.Conn.ReadMessage()
		if err != nil {
			zlog.Error(err.Error())
			return 
		}

		var message = request.ChatMessageRequest{}
		if err := json.Unmarshal(jsonMessage, &message); err != nil {
			zlog.Error(err.Error())
		}
		log.Println("接受消息为：", jsonMessage)

		if messageMode == "channel" {
			for len(ChatServer.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
				sendToMessage := <- c.SendTo
				ChatServer.SendMessageToTransmit(sendToMessage)
			}

			if len(ChatServer.Transmit) < constants.CHANNEL_SIZE {
				ChatServer.SendMessageToTransmit(jsonMessage)
			} else if len(c.SendTo) < constants.CHANNEL_SIZE {
                c.SendTo <- jsonMessage
			} else {
				c.Conn.WriteMessage(websocket.TextMessage, []byte("由于目前同一时间过多用户发送消息，消息发送失败，请稍后重试"))
			}

		} else {
			// kafka模式
			if err := myKafka.KafkaService.ChatWriter.WriteMessages(ctx, kafka.Message{
				Key: []byte(strconv.Itoa(config.GetConfig().KafkaConfig.Partition)),
				Value: jsonMessage,
			}); err != nil {
				zlog.Error(err.Error())
			}
			zlog.Info("已发送信息: " + string(jsonMessage))
		}
	}
}
