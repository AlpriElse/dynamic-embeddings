#!/bin/bash
deploy_spark_setup () {
    scp ./spark_setup.sh $1@$2:~/
    scp -r ./spark_configs $1@$2:~/
    ssh $1@$2 -t "sh ~/spark_setup.sh $1"
    ssh $1@$2 -t "rm -r ~/spark_configs && rm ~/spark_setup.sh"
}

for val in sp21-cs525-g14-0{1..9}.cs.illinois.edu; do
    deploy_spark_setup $1 $val
done
deploy_spark_setup $1 sp21-cs525-g14-10.cs.illinois.edu