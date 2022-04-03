package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gopherlearning/track-devops/cmd/server/handlers"
	"github.com/gopherlearning/track-devops/cmd/server/storage"
	"github.com/gopherlearning/track-devops/cmd/server/web"
)

const (
	serverAddr = "127.0.0.1"
	serverPort = "8080"
)

func main() {
	store := storage.NewStorage()
	h := handlers.NewHandler(store)
	s := web.NewServer(serverAddr+":"+serverPort, h)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	// Периодический вывод содержимого хранилища
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			<-ticker.C

			fmt.Println("==============================")
			for target, values := range store.List() {
				fmt.Printf(`Target "%s":%s`, target, "\n")
				for _, v := range values {
					fmt.Printf("\t%s\n", v)
				}
			}
		}
	}()
	go func() {
		err := s.Start()
		if err != nil {
			fmt.Println(222, err)
		}
	}()
	sig := <-terminate
	s.Shutdown()
	fmt.Printf("Server stoped by signal \"%v\"\n", sig)
}
