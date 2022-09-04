package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gopherlearning/track-devops/internal/repositories"
	"go.uber.org/zap"
)

// showContent период вывода содержимого хранилища
const SHOWCONTENT = 5 * time.Second

// FixAgentArgs исправляет баг в тестах практикума - тесты используют знак = после коротких флагов
func FixAgentArgs() {
	// только для прохождения теста
	for i := 0; i < len(os.Args); i++ {
		if strings.Contains(os.Args[i], "=") {
			a := strings.Split(os.Args[i], "=")
			os.Args[i] = a[1]
			os.Args = append(os.Args[:i], append(a, os.Args[i+1:]...)...)
		}
	}
}

// FixServerArgs исправляет баг в тестах практикума - тесты используют знак = после коротких флагов
func FixServerArgs() {
	// только для прохождения теста
	for i := 0; i < len(os.Args); i++ {
		if strings.Contains(os.Args[i], "=") {
			a := strings.Split(os.Args[i], "=")
			if a[0] == "-r" {
				os.Args[i] = fmt.Sprintf("--restore=%s", a[1])
				continue
			}
			if a[0] == "-d" {
				os.Args[i] = fmt.Sprintf("--database-dsn=%s", a[1])
				continue
			}
			if a[0] == "-crypto-key" {
				os.Args[i] = fmt.Sprintf("--crypto-key=%s", a[1])
				continue
			}
			os.Args = append(os.Args[:i], append(a, os.Args[i+1:]...)...)
		}
	}
}

// ShowStore Периодический вывод содержимого хранилища
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

// InitLogger
func InitLogger(verbose bool) (logger *zap.Logger) {
	if verbose {
		logger, _ = zap.NewDevelopment()
		return
	}
	logger, _ = zap.NewProduction()
	return
}
