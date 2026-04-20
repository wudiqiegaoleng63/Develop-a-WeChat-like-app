package main

import (
    "fmt"
    "kama-chat-server/internal/dao"  // ★导入就会自动执行init()
)

func main() {
    // dao.GormDB已经可用
    fmt.Println("GormDB:", dao.GormDB)
}