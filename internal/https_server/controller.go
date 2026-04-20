package https_server

import (
	"net/http"
    "github.com/gin-gonic/gin"
)

// JsonBack 统一封装响应格式
// ★所有API都用这个函数返回响应
// 参数:
//   c: Gin上下文
//   message: 提示信息（如"登录成功"、"密码错误")
//   ret: 返回状态码（0成功, -2业务失败, -1系统错误）
//   data: 返回的数据（如用户信息）
func JsonBack(c *gin.Context, message string, ret int, data interface{}) {
	if ret == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"message": message,
			"data": data,
		})
	} else if ret == -2 {
		c.JSON(http.StatusOK, gin.H{
			"code": 400,
			"message": message,
		})
	} else if ret == -1 {
		c.JSON(http.StatusOK, gin.H{
			"code": 500,
			"message": message,
		})
	}
}