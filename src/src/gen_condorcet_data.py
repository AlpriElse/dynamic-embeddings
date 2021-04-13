# generates condorcet dataset with N entries for m candidates

import random
import sys

if len(sys.argv) < 4:
    print("Usage: gen_condorcet_data.py num_candidates num_votes output_file")
    exit()
    
m = int(sys.argv[1])
N = int(sys.argv[2])
file_out = sys.argv[3]

candidates = list(range(m))
print(m,N,candidates)
with open(file_out, "w") as outfile:
    for i in range(N):
        random.shuffle(candidates)
        out = [str(i) for i in candidates]
        outfile.write(",".join(out)+"\n")
