import random
from prettytable import PrettyTable

class Node:
    def __init__(self, probe_miss_rate, evil_rec):
        self.probe_miss_rate = probe_miss_rate
        self.evil_rec = evil_rec

        self.ratings = {}


class NetworkSimulator:
    rec_cnt, q_rec_cnt = 3, 2
    probe_cnt, q_probe_cnt = 3, 2
    
    rec_thresh, probe_thresh = 2, 2

    def reset_stats(self):
        self.total_results = 0
        self.good_results = 0
        self.stat_prints_cnt = 0

    def __init__(self, ref_cnt=0):
        self.reset_stats()

        self.nodes : list[Node] = []

        for i in range(ref_cnt):
            info = {j: [5, 0, 5, 0] for j in range(ref_cnt) if j != i}
            self.add_node(info)
    
    def add_node(self, ratings={}, ref_node:int=None, probe_miss_rate=0.0, evil_rec=False):
        ind = len(self.nodes)
        node = Node(probe_miss_rate, evil_rec)
        node.ratings.update(ratings)
        if ref_node is not None:
            node.ratings.update(self.nodes[ref_node].ratings)
            node.ratings[ref_node] = [5, 0, 5, 0]
            self.nodes[ref_node].ratings.setdefault(ind, [0, 0, 0, 0])
        self.nodes.append(node)
    
    def ping(self, node_ind):
        node = self.nodes[node_ind]

        recommenders = []
        q_recommenders = []
        for rec in node.ratings:
            rating = node.ratings[rec]
            if rating[2]-rating[3] >= NetworkSimulator.rec_thresh:
                recommenders.append(rec)
            else:
                q_recommenders.append(rec)
    
        probes = {}
        for rec in random.sample(recommenders, min(len(recommenders), NetworkSimulator.rec_cnt)):
            ratings = self.nodes[rec].ratings
            self.nodes[rec].ratings.setdefault(node_ind, [0, 0, 0, 0])
            for probe in ratings:
                rating = ratings[probe]
                if (probe == node_ind):
                    continue
                probes.setdefault(probe, {"good": False, "recs":[]})
                if rating[0]-rating[1] >= NetworkSimulator.probe_thresh:
                    probes[probe]["good"] = True
                    probes[probe]["recs"].append((rec, False)) # not quarantined
                else:
                    probes[probe]["recs"].append((rec, True)) # quarantined
        for rec in random.sample(q_recommenders, min(len(q_recommenders), NetworkSimulator.q_rec_cnt)):
            ratings = self.nodes[rec].ratings
            self.nodes[rec].ratings.setdefault(node_ind, [0, 0, 0, 0])
            for probe in ratings:
                rating = ratings[probe]
                if (probe == node_ind):
                    continue
                probes.setdefault(probe, {"good": False, "recs":[]})
                if rating[0]-rating[1] >= NetworkSimulator.probe_thresh:
                    probes[probe]["recs"].append((rec, False)) # not quarantined
                else:
                    probes[probe]["recs"].append((rec, True)) # quarantined

        reputable_probes = [probe for probe in probes if probes[probe]["good"]]
        quarantined_probes = [probe for probe in probes if not probes[probe]["good"]]
        
        r_cnt = min(len(reputable_probes), NetworkSimulator.probe_cnt)
        q_cnt = min(len(quarantined_probes), NetworkSimulator.q_probe_cnt)
        picked_probes = random.sample(reputable_probes, r_cnt) + random.sample(quarantined_probes, q_cnt)

        good_answers = 0
        bad_answers = 0
        answers = {}
        for probe in picked_probes:
            self.nodes[probe].ratings.setdefault(node_ind, [0, 0, 0, 0])
            if random.random() < self.nodes[probe].probe_miss_rate:  # bad answer
                answers[probe] = False
                if probes[probe]["good"]:
                    bad_answers += 1
            else:
                answers[probe] = True
                if probes[probe]["good"]:
                    good_answers += 1
        
        self.total_results += 1

        best_ans = good_answers > bad_answers
        if best_ans:
            self.good_results += 1

        for probe in picked_probes:
            node.ratings.setdefault(probe, [0, 0, 0, 0]),
            if answers[probe] == best_ans:
                node.ratings[probe][0] += 1
                for rec in probes[probe]["recs"]:
                    if not rec[1]: # good probe was rated high by recommender
                        node.ratings[rec[0]][2] += 1
            else:
                node.ratings[probe][1] += 1
                for rec in probes[probe]["recs"]:
                    if not rec[1]: # bad probe was rated high by recommender
                        node.ratings[rec[0]][3] += 1

    def print_reputations(self):
        reps = [0]*len(self.nodes)
        for i in range(len(self.nodes)):
            reps[i] = [None]*len(self.nodes)
            for node in self.nodes[i].ratings:
                reps[i][node] = self.nodes[i].ratings[node]
        
        pretty_reps = PrettyTable()
        pretty_reps.field_names = ["ind"] + list(range(len(self.nodes)))
        for i in range(len(self.nodes)):
            pretty_reps.add_row([i] + reps[i])
        
        print(pretty_reps)
    
    def print_stats(self):
        reps = [0]*len(self.nodes)
        for i in range(len(self.nodes)):
            reps[i] = [None]*len(self.nodes)
            for node in self.nodes[i].ratings:
                reps[i][node] = self.nodes[i].ratings[node]
        
        unlinked = 0
        for i in range(len(self.nodes)):
            for j in range(len(self.nodes)):
                if (i == j):
                    continue
                if reps[i][j] is None:
                    unlinked += 1
        
        self.stat_prints_cnt += 1
        print(f"Stat {self.stat_prints_cnt}")
        print(f"Total pings: {self.total_results}")
        print(f"Accuracy: {self.good_results/self.total_results if self.total_results != 0 else None}")
        print(f"Missing links (# | %): {unlinked} | {unlinked/(len(self.nodes)*len(self.nodes)-len(self.nodes))}")
        print("-------------------")


if __name__ == "__main__":
    s = NetworkSimulator(5)
    for ref in range(5):
        for i in range(4):
            s.add_node(ref_node=ref, probe_miss_rate=1)
    
    s.print_stats()

    for step in range(100):
        for node in range(len(s.nodes)):
            s.ping(node)
    
    s.print_stats()
        
