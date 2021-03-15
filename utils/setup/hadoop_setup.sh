local_setup () {
    # Install
    wget https://downloads.apache.org/hadoop/common/hadoop-3.2.2/hadoop-3.2.2.tar.gz
    sudo tar zvxf hadoop-3.2.2.tar.gz -C /opt
    sudo rm hadoop-3.2.2.tar.gz
    cd /opt
    sudo rm -r hadoop # cleanup
    sudo mv hadoop-3.2.2 hadoop
    ##sudo chown jit2:jit2 -R hadoop
    sudo chgrp -R csvm525-stu /opt/hadoop
    sudo chmod 770 -R /opt/hadoop
    # Setup environment variables
    echo "export HADOOP_HOME=/opt/hadoop" >> ~/.bashrc
    echo "export PATH=\$PATH:\$HADOOP_HOME/bin:\$HADOOP_HOME/sbin" >> ~/.bashrc
    echo "export JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.282.b08-1.el7_9.x86_64" >> /opt/hadoop/etc/hadoop/hadoop-env.sh
    source ~/.bashrc

    # Copy hosts file
    sudo cp ~/hadoop_configs/hosts /etc/

    # Create directories for HDFS namenode/datanode
    mkdir -p /opt/hadoop/data/hdfs/datanode
    mkdir -p /opt/hadoop/data/hdfs/namenode
    ## Copy hadoop configs 
    rm /opt/hadoop/etc/hadoop/hdfs-site.xml
    cp ~/hadoop_configs/hdfs-site.xml /opt/hadoop/etc/hadoop/
    rm /opt/hadoop/etc/hadoop/yarn-site.xml
    cp ~/hadoop_configs/yarn-site.xml /opt/hadoop/etc/hadoop/
    rm /opt/hadoop/etc/hadoop/core-site.xml
    cp ~/hadoop_configs/core-site.xml /opt/hadoop/etc/hadoop/
    rm /opt/hadoop/etc/hadoop/mapred-site.xml
    cp ~/hadoop_configs/mapred-site.xml /opt/hadoop/etc/hadoop/
    rm /opt/hadoop/etc/hadoop/workers
    cp ~/hadoop_configs/workers /opt/hadoop/etc/hadoop/

    # Format and boot HDFS
    # hdfs namenode -format -force
    # start-dfs.sh
    # To view: ssh -L local_port:localhost:remote_port ssh host
    # Ex: ssh -L 9870:localhost:9870 jit2@sp21-cs525-g14-01.cs.illinois.edu
}

local_setup $1
