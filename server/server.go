package server

import (
	"context"
	"errors"
	"fmt"
	"log" // TODO: remove log from server code
	"net"
	"net/http"

	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/reputation"

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

type PingerServer struct {
	RepManager *reputation.ReputationManager
	pb.UnimplementedPingerServer
}

func (s *PingerServer) CheckHost(ctx context.Context, in *pb.CheckHostRequest) (*pb.CheckHostResponse, error) {
	sender := in.GetSender()
	if s.RepManager.GetReputation(sender) <= 0 {
		return &pb.CheckHostResponse{Code: -1}, errors.New(fmt.Sprintf("not enough reputation for node %s to use this probe", sender))
	}

	res, err := check(in.GetHost())
	s.RepManager.LowerClient(sender)
	if err != nil {
		return &pb.CheckHostResponse{Code: -1}, err // TODO: probably should not return all server-side errors to client
	}
	return &pb.CheckHostResponse{Code: int32(res)}, nil
}

func (pingerServer *PingerServer) Serve(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPingerServer(s, pingerServer)

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
