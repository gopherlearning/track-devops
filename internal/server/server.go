package server

import (
	"fmt"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/internal/server/rpc"
	"github.com/gopherlearning/track-devops/internal/server/web"
	"go.uber.org/zap"
)

type Server interface {
	Stop() error
}

func NewServer(args *internal.ServerArgs, store repositories.Repository) (s Server, err error) {
	switch args.Transport {
	case "http":
		s, err = web.NewEchoServer(store, args.ServerAddr, args.Verbose, web.WithKey([]byte(args.Key)), web.WithPprof(args.UsePprof), web.WithLogger(zap.L()), web.WithCryptoKey(args.CryptoKey), web.WithTrustedSubnet(args.TrustedSubnet))
		if err != nil {
			return nil, err
		}
		return s, nil
	case "grpc":
		s, err = rpc.NewGrpcServer(store, args.ServerAddr, args.Verbose, rpc.WithKey([]byte(args.Key)), rpc.WithLogger(zap.L()), rpc.WithTrustedSubnet(args.TrustedSubnet))
		if err != nil {
			return nil, err
		}
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported trunsport type: %s", args.Transport)
	}
}
