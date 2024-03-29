package rpc

import (
	"context"
	"fmt"
	"net"

	"github.com/gopherlearning/track-devops/internal/metrics"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/proto"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type RPCServer struct {
	trusted  *net.IPNet
	s        repositories.Repository
	g        *grpc.Server
	servOpts []grpc.ServerOption
	logger   *zap.Logger
	key      []byte
	proto.UnimplementedMonitoringServer
}

var _ proto.MonitoringServer = (*RPCServer)(nil)

// RPCServerOptionFunc определяет тип функции для опций.
type RPCServerOptionFunc func(*RPCServer)

// WithKey задаёт ключ для подписи
func WithKey(key []byte) RPCServerOptionFunc {
	return func(s *RPCServer) {
		s.key = key
	}
}

// WithTrustedSubnet задаёт сеть доверенных адресов агентов
func WithTrustedSubnet(trusted string) RPCServerOptionFunc {
	return func(s *RPCServer) {
		if len(trusted) == 0 {
			return
		}
		_, trusted, err := net.ParseCIDR(trusted)
		if err != nil {
			if s.logger != nil {
				s.logger.Error(err.Error())
			}
			return
		}
		s.trusted = trusted
		s.servOpts = append(s.servOpts,
			grpc_middleware.WithUnaryServerChain(
				func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
					p, ok := peer.FromContext(ctx)
					if !ok {
						return nil, status.Error(codes.InvalidArgument, "access denied, no header")
					}
					realIP, _, err := net.SplitHostPort(p.Addr.String())
					if err != nil {
						return nil, status.Error(codes.InvalidArgument, "адрес не определён")
					}
					ip := net.ParseIP(realIP)
					if ip == nil {
						return nil, status.Error(codes.InvalidArgument, "access denied, bad ip")
					}
					if !s.trusted.Contains(ip) {
						return nil, status.Error(codes.PermissionDenied, "access denied")
					}
					return handler(ctx, req)
				},
			),
			grpc_middleware.WithStreamServerChain(
				func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
					p, ok := peer.FromContext(stream.Context())
					if !ok {
						return status.Error(codes.InvalidArgument, "access denied, no header")
					}
					realIP, _, err := net.SplitHostPort(p.Addr.String())
					if err != nil {
						return status.Error(codes.InvalidArgument, "access denied, bad ip")
					}
					ip := net.ParseIP(realIP)
					if ip == nil {
						return status.Error(codes.InvalidArgument, "access denied, bad ip")
					}
					if !s.trusted.Contains(ip) {
						return status.Error(codes.PermissionDenied, "access denied")
					}
					return handler(srv, stream)
				},
			))
	}
}

// WithLogger set logger
func WithLogger(logger *zap.Logger) RPCServerOptionFunc {
	return func(s *RPCServer) {
		s.logger = logger
	}
}

func NewRPCServer(store repositories.Repository, listen string, debug bool, opts ...RPCServerOptionFunc) (*RPCServer, error) {
	servOpts := make([]grpc.ServerOption, 0)
	if !debug {
		servOpts = append(servOpts,
			grpc_middleware.WithUnaryServerChain(
				grpc_recovery.UnaryServerInterceptor(),
			),
			grpc_middleware.WithStreamServerChain(
				grpc_recovery.StreamServerInterceptor(),
			),
		)
	}
	serv := &RPCServer{
		s:        store,
		servOpts: servOpts,
	}

	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("option error: %v", opt)
		}
		opt(serv)
	}
	RPCServer := grpc.NewServer(serv.servOpts...)
	proto.RegisterMonitoringServer(RPCServer, serv)
	serv.g = RPCServer
	if len(listen) != 0 {
		go func() {
			lis, err := net.Listen("tcp", listen)
			if err != nil {
				serv.logger.Error(err.Error())
				return
			}
			defer lis.Close()
			if err := RPCServer.Serve(lis); err != nil {
				serv.logger.Error(err.Error())
			}
		}()
	}
	return serv, nil
}

// Update ...
func (s *RPCServer) Update(ctx context.Context, req *proto.UpdateRequest) (*proto.Empty, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "адрес не определён")
	}
	realIP, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "access denied, bad ip")
	}
	for _, v := range req.Metrics {
		err = s.saveMetric(v, realIP)
		if err != nil {
			return nil, err
		}
	}
	return &proto.Empty{}, nil
}

// Updates ...
// func (s *RPCServer) Updates(stream proto.Monitoring_UpdatesServer) (err error) {
// 	p, ok := peer.FromContext(stream.Context())
// 	if !ok {
// 		return status.Error(codes.InvalidArgument, "адрес не определён")
// 	}
// 	realIP, _, err := net.SplitHostPort(p.Addr.String())
// 	if err != nil {
// 		return status.Error(codes.InvalidArgument, "адрес не определён")
// 	}
// 	var req *proto.Metric
// 	for {
// 		req, err = stream.Recv()
// 		if err == io.EOF {
// 			return stream.SendAndClose(&proto.Empty{})
// 		}
// 		if err != nil {
// 			return status.Error(codes.InvalidArgument, err.Error())
// 		}
// 		_, err = s.saveMetric(req, realIP)
// 		if err != nil {
// 			return err
// 		}
// 	}
// }

func (s *RPCServer) GetMetric(ctx context.Context, req *proto.MetricRequest) (*proto.Metric, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "адрес не определён")
	}
	realIP, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "адрес не определён")
	}
	if len(protoTypeToMetricType(req.GetType())) == 0 {
		return nil, status.Error(codes.InvalidArgument, repositories.ErrWrongMetricType.Error())
	}
	m, err := s.s.GetMetric(ctx, realIP, protoTypeToMetricType(req.GetType()), req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	resp := &proto.Metric{
		Id:   m.ID,
		Hash: m.Hash,
		Type: GetMetricProtoType(m),
	}
	switch resp.Type {
	case proto.Type_COUNTER:
		resp.Value = &proto.Metric_Counter{Counter: *m.Delta}
	case proto.Type_GAUGE:
		resp.Value = &proto.Metric_Gauge{Gauge: *m.Value}
	default:
		return nil, status.Error(codes.InvalidArgument, repositories.ErrWrongMetricType.Error())
	}
	return resp, nil
}

func (s *RPCServer) Ping(ctx context.Context, req *proto.Empty) (*proto.Empty, error) {
	if err := s.s.Ping(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &proto.Empty{}, nil
}

func (s *RPCServer) Stop() error {
	s.g.GracefulStop()
	return nil
}

func (s *RPCServer) saveMetric(req *proto.Metric, peer string) error {
	m := metrics.Metrics{
		ID:   req.Id,
		Hash: req.Hash,
	}
	switch req.GetType() {
	case proto.Type_COUNTER:
		m.MType = metrics.CounterType
		v := req.GetCounter()
		m.Delta = &v
	case proto.Type_GAUGE:
		m.MType = metrics.GaugeType
		v := req.GetGauge()
		m.Value = &v
	default:
		return status.Error(codes.InvalidArgument, repositories.ErrWrongMetricType.Error())
	}
	if len(s.key) != 0 {
		recived := m.Hash
		err := m.Sign(s.key)
		if err != nil || recived != m.Hash {
			return status.Error(codes.InvalidArgument, "подпись не соответствует ожиданиям")
		}
	}
	if err := s.s.UpdateMetric(context.TODO(), peer, m); err != nil {
		switch err {
		case repositories.ErrWrongMetricURL:
			return status.Error(codes.NotFound, err.Error())
		case repositories.ErrWrongMetricValue:
			return status.Error(codes.InvalidArgument, err.Error())
		case repositories.ErrWrongValueInStorage:
			return status.Error(codes.Unimplemented, err.Error())
		default:
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}

func protoTypeToMetricType(t proto.Type) metrics.MetricType {
	switch t {
	case proto.Type_COUNTER:
		return metrics.CounterType
	case proto.Type_GAUGE:
		return metrics.GaugeType
	default:
		return ""
	}
}

func GetMetricProtoType(m *metrics.Metrics) proto.Type {
	switch m.MType {
	case metrics.CounterType:
		return proto.Type_COUNTER
	case metrics.GaugeType:
		return proto.Type_GAUGE
	default:
		return proto.Type_UNKNOWN
	}
}
