package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	pb "github.com/rybbba/dist-pinger/pinger"

	"google.golang.org/grpc"
)

func check(host string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s", host))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	pb.UnimplementedPingerServer
}

func (s *server) CheckHost(ctx context.Context, in *pb.CheckHostRequest) (*pb.CheckHostResponse, error) {
	res, err := check(in.GetHost())
	if err != nil {
		return &pb.CheckHostResponse{Code: -1}, err
	}
	return &pb.CheckHostResponse{Code: int32(res)}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPingerServer(s, &server{})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
