package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func JsonBack(c *gin.Context, message string, ret int, data interface{}) {
	if ret == 0 {
		if data != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": message,
				"data":    data,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": message,
			})
		}
	} else if ret == -2 {
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": message,
		})
	} else if ret == -1 {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": message,
		})
	}
}

// GetTokenUuid 从JWT中间件注入的上下文中获取当前用户uuid
func GetTokenUuid(c *gin.Context) string {
	uuid, exists := c.Get("uuid")
	if !exists {
		return ""
	}
	s, ok := uuid.(string)
	if !ok {
		return ""
	}
	return s
}

// CheckOwner 校验请求中的ownerId是否与token中的uuid一致
// 返回true表示一致（通过），false表示不一致（拒绝）
func CheckOwner(c *gin.Context, ownerId string) bool {
	tokenUuid := GetTokenUuid(c)
	if tokenUuid != ownerId {
		c.JSON(http.StatusOK, gin.H{
			"code":    403,
			"message": "无权操作他人数据",
		})
		return false
	}
	return true
}