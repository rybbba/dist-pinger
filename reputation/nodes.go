package reputation

import "github.com/rybbba/dist-pinger/identity"

var (
	reputationThreshold  = 2
	credibilityThreshold = 2
)

type Node struct {
	user            identity.PublicUser
	reputationGood  int
	reputationBad   int
	credibilityGood int
	credibilityBad  int
}

func nodeInit(user identity.PublicUser) Node {
	return Node{user: user} // all other fields will be zero by default
}

func nodeInitRef(user identity.PublicUser) Node {
	node := nodeInit(user)
	node.reputationGood = 5
	node.credibilityGood = 5
	return node
}

func IsReputable(node Node) bool {
	return node.reputationGood-node.reputationBad >= reputationThreshold
}

func RaiseReputation(node Node) Node {
	node.reputationGood += 1
	return node
}

func LowerReputation(node Node) Node {
	node.reputationBad += 1
	return node
}

func IsCredible(node Node) bool {
	return node.credibilityGood-node.credibilityBad >= credibilityThreshold
}

func RaiseCredibility(node Node) Node {
	node.credibilityGood += 1
	return node
}

func LowerCredibility(node Node) Node {
	node.credibilityBad += 1
	return node
}

type probeRecommender struct {
	user             identity.PublicUser
	quarantinedProbe bool
}

type Probe struct {
	User      identity.PublicUser
	Reputable bool

	recommenders []probeRecommender
}
