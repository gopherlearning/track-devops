package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/gopherlearning/track-devops/internal/repositories"
	"go.uber.org/zap"
)

// SHOWCONTENT период вывода содержимого хранилища
const SHOWCONTENT = 5 * time.Second

// ShowStore периодически выводит содержимое хранилища
func ShowStore(store repositories.Repository, logger *zap.Logger) {
	ticker := time.NewTicker(SHOWCONTENT)
	for {
		<-ticker.C

		fmt.Println("==============================")
		var list map[string][]string
		list, err := store.List(context.Background())
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		for target, values := range list {
			fmt.Printf(`Target "%s":%s`, target, "\n")
			for _, v := range values {
				fmt.Printf("\t%s\n", v)
			}
		}
	}
}

// InitLogger возвращает логер
func InitLogger(verbose bool) (logger *zap.Logger) {
	if verbose {
		logger, _ = zap.NewDevelopment()
		return
	}
	logger, _ = zap.NewProduction()
	zap.ReplaceGlobals(logger)
	return
}
