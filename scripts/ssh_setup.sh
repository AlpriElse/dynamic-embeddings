#!/bin/bash
ssh_firsttime () {
    for val in sp21-cs525-g14-0{2..9}.cs.illinois.edu; do
        echo "$1@$val"
        sudo ssh $1@$val "ls"
    done
    sudo ssh $1@sp21-cs525-g14-10.cs.illinois.edu "ls"
}

setup_git_credentials () {
    for val in sp21-cs525-g14-0{0..1}.cs.illinois.edu; do
        sudo ssh $1@$val "git config --global credential.helper store"
    done
    sudo ssh $1@sp21-cs525-g14-10.cs.illinois.edu "git config --global credential.helper store"
}

setup_ssh_key () {
    ssh-keygen -t rsa
    for val in sp21-cs525-g14-0{0..1}.cs.illinois.edu; do
        ssh-copy-id -i /home/$1/.ssh/id_rsa.pub $1@$val
    done
    ssh-copy-id -i /home/$1/.ssh/id_rsa.pub $1@sp21-cs525-g14-10.cs.illinois.edu
}
#ssh_firsttime $1
setup_ssh_key $1
setup_git_credentials $1

