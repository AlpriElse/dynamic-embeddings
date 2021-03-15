get_scala () {
    wget https://downloads.lightbend.com/scala/2.13.5/scala-2.13.5.rpm
    sudo yum -y install scala-2.13.5.rpm
    scala -version
    rm scala-2.13.5.rpm
}

get_scala

wget https://downloads.apache.org/spark/spark-3.1.1/spark-3.1.1-bin-hadoop3.2.tgz
sudo tar -xvf spark-3.1.1-bin-hadoop3.2.tgz -C /opt
rm spark-3.1.1-bin-hadoop3.2.tgz
cd /opt
sudo rm -r spark
sudo mv spark-3.1.1-bin-hadoop3.2 spark
# Permissions stuff
sudo chgrp -R csvm525-stu /opt/spark
sudo chmod 770 -R /opt/spark

# Environment stuff
cp /opt/spark/conf/spark-env.sh{.template,}
# Set up environment variables
echo "export SPARK_HOME=/opt/spark" >> ~/.bashrc
echo "export PATH=\$PATH:\$SPARK_HOME/bin" >> ~/.bashrc
echo "export SPARK_MASTER_HOST=node-master" >> /opt/spark/conf/spark-env.sh
echo "export JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.282.b08-1.el7_9.x86_64" >> /opt/spark/conf/spark-env.sh
rm /opt/spark/conf/workers
cp ~/spark_configs/workers /opt/spark/conf/workers
source ~/.bashrc

# Start master: spark-2.2.3-bin-hadoop2.7/sbin/start-master.sh
# Start follower: spark-2.2.3-bin-hadoop2.7/sbin/start-slave.sh 
# To start: spark-2.2.3-bin-hadoop2.7/sbin/start-all.sh
# To stop: spark-2.2.3-bin-hadoop2.7/sbin/stop-all.sh


#export JAVA_HOME=//jdk1.8.0_201
#export PATH=${JAVA_HOME}/bin:$PATH"

