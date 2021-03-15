#!/bin/bash
#wget https://repo.anaconda.com/archive/Anaconda3-2020.11-Linux-x86_64.sh
#bash Anaconda3-2020.11-Linux-x86_64.sh
#conda install -c conda-forge tensorboard
#conda init bash
#conda create -n pytorch_env pytorch torchvision torchaudio -c pytorch
#Open environment: conda activate [netid]_pytorch

setup_pytorch_all () {
    for val in sp21-cs525-g14-0{1..9}.cs.illinois.edu; do
        setup_pytorch $1 $val
    done
    setup_pytorch $1 sp21-cs525-g14-10.cs.illinois.edu
}

setup_pytorch () {
    ssh $1@$2 "wget https://repo.anaconda.com/archive/Anaconda3-2020.11-Linux-x86_64.sh"
    #sudo scp Anaconda3-2020.11-Linux-x86_64.sh $1@$2:~/Anaconda3-2020.11-Linux-x86_64.sh
    ssh $1@$2 "bash Anaconda3-2020.11-Linux-x86_64.sh -b && 
    ./anaconda3/bin/conda init bash &&
    ./anaconda3/bin/conda install -y -c conda-forge tensorboard && 
    ./anaconda3/bin/conda create -y -n $1_pytorch pytorch torchvision torchaudio -c pytorch"
    ssh $1@$2 "rm Anaconda3-2020.11-Linux-x86_64.sh"
}

rm_pytorch_all () {
    for val in sp21-cs525-g14-0{1..9}.cs.illinois.edu; do
        rm_pytorch $1 $val
    done
    rm_pytorch $1 sp21-cs525-g14-10.cs.illinois.edu
}

rm_pytorch () {
    ssh $1@$2 "rm Anaconda3-2020.11-Linux-x86_64.sh"
    ssh $1@$2 "./anaconda3/bin/conda install anaconda-clean -y && anaconda-clean -y && rm -rf ~/anaconda3"
}

setup_pytorch_all $1
#rm_pytorch_all $1
