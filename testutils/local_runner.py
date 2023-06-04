import select
import subprocess
from subprocess import PIPE, STDOUT
import pty
from time import sleep
import threading
from typing import IO

# OUTDATED

class Node:
    def __init__(self, proc: subprocess.Popen, inp: IO[str], out: IO[str]):
        raise NotImplementedError("Outdated node wrapping, public key handling is not implemented.")
        self.p = proc

        self.input_handler = inp
        self.output_handler = out

        self.poller = select.poll()
        self.poller.register(self.output_handler, select.POLLIN)

    def getReps(self):
        while self.poller.poll(1):
            self.output_handler.read(128)
        self.input_handler.write("r\n")
        self.input_handler.flush()
        ans = self.output_handler.readline()

        res = {}
        for ent in ans.split(","):
            address, rep, cred = ent.split()
            res[address] = (rep, cred)
        return res
    
    def ping(self, host="google.com"):
        self.input_handler.write(f"{host}\n")

def run_local_ref_node(port, low=5001, high=5010):
    args = ["go", "run", ".", "--id", f"localhost:{port}", "--port", str(port)]
    for i in range(low, high):
        if i != port:
            args.append(f"localhost:{i}")
    p = subprocess.Popen(args, text=True, stdin=PIPE, stdout=PIPE, stderr=PIPE)
    return Node(p, p.stdin, p.stdout)


def run_local_newbie_node(port, ref_port):
    args = ["go", "run", ".", "--id", f"localhost:{port}", "--port", str(port), "--ref", f"localhost:{ref_port}"]
    p = subprocess.Popen(args, text=True, stdin=PIPE, stdout=PIPE, stderr=PIPE)
    return Node(p, p.stdin, p.stdout)


class NodeStat:
    def __init__(self) -> None:
        self.known_by = 0
        self.reps = []
        self.creds = []

class NodeGroup:
    def __init__(self) -> None:
        self.nodes : list[Node] = []
        self.iter = 1

    def add_node(self, node):
        self.nodes.append(node)

    def step(self, delay=3):
        sleep(delay)

        stats = {}
        for node in self.nodes:
            reps = node.getReps()
            for addr in reps:
                rep, cred = reps[addr]

                stats.setdefault(addr, NodeStat())
                stats[addr].known_by += 1
                stats[addr].reps.append(rep)
                stats[addr].creds.append(cred)
        
        print(f"Step {self.iter}")
        for node in sorted(list(stats.items()), key=lambda x: x[0]):
            print(node[0], node[1].known_by, node[1].reps, node[1].creds)
        print("-------")
        self.iter += 1



g = NodeGroup()
for i in range(5001, 5010):
    g.add_node(run_local_ref_node(i))

g.step()

curious = run_local_newbie_node(5010, 5003)
g.add_node(curious)

g.step()

for node in g.nodes:
    node.ping()

g.step(5)

for node in g.nodes:
    node.ping()

g.step(5)
