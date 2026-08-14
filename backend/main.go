package main

import (
	"log"

	"galgame-maker/internal/api"
	"galgame-maker/internal/config"
	"galgame-maker/internal/database"
)

func main() {
	cfg := config.Load()

	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	r := api.NewRouter(db, cfg)
	addr := ":" + cfg.Port
	log.Printf("Galgame Maker 后端已启动: http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
