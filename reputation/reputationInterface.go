package reputation

import (
	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/identity"
)

type ReputationManagerInterface interface {
	InitNodes(users []identity.PublicUser)

	GetProbes(sender identity.PrivateUser, pickProbes int) []Probe

	CopyReputation(sender identity.PrivateUser, target identity.PublicUser) error

	EvaluateVotes(probes []Probe, satisfaction []int)

	PrintSimpleRep() string                                                                 // Debug function
	GiveProbes(sender identity.PublicUser, withCredibility bool) *pb.GetReputationsResponse // not very interface-like, should probably be refactored
}
