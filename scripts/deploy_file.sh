for val in sp21-cs525-g14-0{2..9}.cs.illinois.edu; do
	sudo ssh $1@$val "sudo rm -r $2"
	sudo scp $2 $1@$val:$2
done

sudo ssh $1@sp21-cs525-g14-10.cs.illinois.edu "sudo rm -r $2"
sudo scp $2 $1@sp21-cs525-g14-10.cs.illinois.edu:$2