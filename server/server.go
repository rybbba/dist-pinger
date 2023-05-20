package server

import (
	"context"
	"fmt"
	"log" // TODO: remove log from server code
	"net"
	"net/http"

	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/identity"
	"github.com/rybbba/dist-pinger/reputation"

	"google.golang.org/grpc"
)

func check(host string) (int, error) {
	// TODO: add host string verification

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
	pb.UnimplementedReputationServer
}

func (s *PingerServer) GetReputations(ctx context.Context, in *pb.GetReputationsRequest) (*pb.GetReputationsResponse, error) {
	sender := in.GetSender()
	senderUser, err := identity.ParseUser(sender)
	if err != nil {
		return &pb.GetReputationsResponse{}, err
	}

	needCredibilities := in.GetNeedCredibilities()

	return s.RepManager.GiveProbes(senderUser, needCredibilities), nil
}

func (s *PingerServer) CheckHost(ctx context.Context, in *pb.CheckHostRequest) (*pb.CheckHostResponse, error) {
	sender := in.GetSender()
	_, err := identity.ParseUser(sender)
	if err != nil {
		return &pb.CheckHostResponse{}, err
	}

	res, err := check(in.GetHost())
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
	pb.RegisterReputationServer(s, pingerServer)

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
