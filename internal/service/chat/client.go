package chat

import (
	"context"
	"encoding/json"
	"log"

	"kama-chat-server/internal/config"
	"kama-chat-server/internal/dto/request"
	myKafka "kama-chat-server/internal/service/kafka"
	"kama-chat-server/pkg/zlog"
	"net/http"

	"github.com/gorilla/websocket"
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
			
		}
	}
}
