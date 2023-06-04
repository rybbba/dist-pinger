package server

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/identity"
	"github.com/rybbba/dist-pinger/reputation"

	"google.golang.org/grpc"
)

type PingerServer struct {
	RepManager reputation.ReputationManagerInterface
	user       identity.PrivateUser
	pb.UnimplementedPingerServer
	pb.UnimplementedReputationServer
}

func (s *PingerServer) SetUser(user identity.PrivateUser) {
	s.user = user
}

func (s *PingerServer) GetReputations(ctx context.Context, in *pb.GetReputationsRequest) (*pb.GetReputationsResponse, error) {
	sender := in.GetSender()
	senderUser, err := identity.ParseUser(sender)
	if err != nil {
		return &pb.GetReputationsResponse{}, err
	}
	signature := in.Signature
	in.Signature = nil
	err = identity.VerifyProto(senderUser, in, signature)
	if err != nil {
		return &pb.GetReputationsResponse{}, err
	}

	needCredibilities := in.GetNeedCredibilities()

	messageP := s.RepManager.GiveProbes(senderUser, needCredibilities)
	signature, err = identity.SignProto(s.user, messageP)
	if err != nil {
		log.Fatalf("cannot sign message: %v", err)
	}
	messageP.Signature = signature
	return messageP, nil
}

func (s *PingerServer) CheckHost(ctx context.Context, in *pb.CheckHostRequest) (*pb.CheckHostResponse, error) {
	sender := in.GetSender()
	senderUser, err := identity.ParseUser(sender)
	if err != nil {
		return &pb.CheckHostResponse{}, err
	}
	signature := in.Signature
	in.Signature = nil
	err = identity.VerifyProto(senderUser, in, signature)
	if err != nil {
		return &pb.CheckHostResponse{}, err
	}

	res, err := check(in.GetHost())
	if err != nil {
		return &pb.CheckHostResponse{Code: -1}, err // TODO: probably should not return all server-side errors to client
	}

	message := pb.CheckHostResponse{Code: int32(res)}
	signature, err = identity.SignProto(s.user, &message)
	if err != nil {
		log.Fatalf("cannot sign message: %v", err)
	}
	message.Signature = signature
	return &message, nil
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
