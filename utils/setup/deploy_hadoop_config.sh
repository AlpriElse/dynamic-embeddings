sudo rm /opt/hadoop/etc/hadoop/$2
sudo cp ./$2 /opt/hadoop/etc/hadoop/

for val in sp21-cs525-g14-0{2..9}.cs.illinois.edu; do
	sudo ssh $1@$val "sudo rm -r /opt/hadoop/etc/hadoop/$2"
	sudo scp ./$2 $1@$val:/hadoop/etc/hadoop
done

sudo ssh $1@sp21-cs525-g14-10.cs.illinois.edu "sudo rm -r /opt/hadoop/etc/hadoop/$2"
sudo scp ./$2 $1@sp21-cs525-g14-10.cs.illinois.edu:/opt/hadoop/etc/hadoop