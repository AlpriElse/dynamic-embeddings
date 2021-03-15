# dynamic-embeddings

### Deployment

To deploy your branch to all nodes
1. Git commit, push your branch to github.

2. From repository root directory: 
`sh utils/deploy_branch.sh [netid] [branch name]`

## Cluster Usage

To launch or manage VMs: https://vc.cs.illinois.edu/ui/

### Apache Hadoop

Start HDFS cluster (10 nodes) without YARN
```
start-dfs.sh
```

With YARN (job scheduling/MapReduce)
```
start-yarn.sh
```

Stopping cluster
```
stop-all.sh
```

Monitor cluster health, ssh using
```
ssh -L 9870:localhost:9870 [netid]@sp21-cs525-g14-01.cs.illinois.edu
```
Go to `http://localhost:9870` in the browser.

[Hadoop Cluster Documentation](https://hadoop.apache.org/docs/current/hadoop-project-dist/hadoop-common/ClusterSetup.html)

### Apache Spark

Start Spark standalone cluster on all workers
```
/opt/spark/sbin/start-all.sh
```

Start only master
```
/opt/spark/sbin/start-master.sh
```

Start worker on worker node
```
/opt/spark/sbin/start-worker.sh <master-spark-URL>
```

Monitor cluster health, ssh using
```
ssh -L 8080:localhost:8080 [netid]@sp21-cs525-g14-01.cs.illinois.edu
```
Go to `http://localhost:8080` in the browser.

[Spark Cluster Documentation](https://spark.apache.org/docs/latest/spark-standalone.html)

### PyTorch Distributed

`conda activate [netid]_pytorch`

[PyTorch Distributed Documentation](https://pytorch.org/tutorials/beginner/dist_overview.html)
