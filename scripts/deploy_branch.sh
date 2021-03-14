#!/bin/bash
# Make sure to run git config --global credential.helper store once to save username/pass
# Usage: ./deploy_branch.sh [netid] [branch name]
for val in sp21-cs525-g14-0{2..9}.cs.illinois.edu; do
   if ssh $1@$val '[ ! -d ~/dynamic-embeddings ]'
   then
       ssh $1@$val -t 'git clone https://github.com/AlpriElse/dynamic-embeddings.git'
   else 
       ssh $1@$val -t "cd dynamic-embeddings && git fetch && git pull && git checkout $2"
   fi
done

if ssh $1@sp21-cs525-g14-10.cs.illinois.edu '[ ! -d ~/dynamic-embeddings ]'
then
    ssh $1@sp21-cs525-g14-10.cs.illinois.edu -t 'git clone https://github.com/AlpriElse/dynamic-embeddings.git'
else
    ssh $1@sp21-cs525-g14-10.cs.illinois.edu -t "cd dynamic-embeddings && git fetch && git pull && git checkout $2"
fi
