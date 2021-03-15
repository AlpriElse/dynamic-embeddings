#!/bin/bash
deploy_hadoop_setup () {
    scp ./hadoop_setup.sh $1@$2:~/
    scp -r ./hadoop_configs $1@$2:~/
    ssh $1@$2 -t "sh ~/hadoop_setup.sh $1"
    ssh $1@$2 -t "rm -r ~/hadoop_configs && rm ~/hadoop_setup.sh"
}

for val in sp21-cs525-g14-0{3..4}.cs.illinois.edu; do
    deploy_hadoop_setup $1 $val
done
deploy_hadoop_setup $1 sp21-cs525-g14-10.cs.illinois.edu

#sh ./deploy_hadoop_config.sh $1 yarn-site.xml
#sh ./deploy_hadoop_config.sh $1 core-site.xml
#sh ./deploy_hadoop_config.sh $1 hdfs-site.xml
#sh ./deploy_hadoop_config.sh $1 mapred-site.xml
#sh ./deploy_hadoop_config.sh $1 workers
##sh ./deploy_hadoop_config.sh $1 master//srek
#sh ./deploy_hadoop_config.sh $1 workers
##sh ./deploy_hadoop_config.sh $1 master
#
## Deploy to other machines
#sh ./deploy_file.sh $1 /home/$1/hadoop-3.2.2.tar.gz