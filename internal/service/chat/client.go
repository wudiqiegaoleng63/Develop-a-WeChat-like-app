package chat

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"kama-chat-server/internal/config"
	"kama-chat-server/internal/dao"
	"kama-chat-server/internal/dto/request"
	"kama-chat-server/internal/model"
	myKafka "kama-chat-server/internal/service/kafka"
	"kama-chat-server/pkg/constants"
	"kama-chat-server/pkg/enum/message/message_status_enum"
	"kama-chat-server/pkg/zlog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)


// MessageBack 返回给前端的消息结构
type MessageBack struct {
	Message []byte
	Uuid    string
}

// Client WebSocket客户端结构体
type Client struct {
	Conn     *websocket.Conn // WebSocket连接
	Uuid     string          // 用户唯一标识
	SendTo   chan []byte     // 给Server端的Channel（Channel模式缓冲）
	SendBack chan *MessageBack // 发送给前端的Channel
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
		// 强制使用认证后的UUID作为发送者，防止伪造身份
		message.SendId = c.Uuid
		// 重新序列化，确保使用覆盖后的send_id
		if overridden, err := json.Marshal(message); err == nil {
			jsonMessage = overridden
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


func (c *Client) Write() {
	zlog.Info("ws write goroutine start")
	for messageBack := range c.SendBack {
		err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
		if err != nil {
			zlog.Error(err.Error())
			return
		}

		if res := dao.GormDB.Model(&model.Message{}).Where("uuid=?", messageBack.Uuid).Update("status", message_status_enum.Sent); res.Error != nil {
			zlog.Error(res.Error.Error())
		}
	}
}


// ============================================================
// NewClientInit - 创建并初始化Client
// ============================================================

func NewClientInit(c *gin.Context, clientId string) {
	kafkaConfig := config.GetConfig().KafkaConfig

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zlog.Error(err.Error())
	}

	client := &Client{
		Conn:	conn,
		Uuid:   clientId,
		SendTo: make(chan []byte, constants.CHANNEL_SIZE),
		SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
	}

	if kafkaConfig.MessageMode == "channel" {
		ChatServer.SendClientToLogin(client)
	} else {
		KafkaChatServer.SendClientToLogin(client)
	}

	go client.Read()
	go client.Write()
	zlog.Info("ws连接成功")
}


// ============================================================
// ClientLogout - WebSocket客户端登出
// ============================================================
// ★当前端用户退出登录时调用
func ClientLogout(clientId string) (string, int) {
	kafkaConfig := config.GetConfig().KafkaConfig

	client := ChatServer.Clients[clientId]

	if client != nil {
		if kafkaConfig.MessageMode == "channel" {
			ChatServer.SendClientToLogout(client)
		} else {
			KafkaChatServer.SendClientToLogout(client)
		}

		if err := client.Conn.Close(); err != nil {
			zlog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}

		close(client.SendTo)
		close(client.SendBack)
	}

	return "退出成功", 0
}