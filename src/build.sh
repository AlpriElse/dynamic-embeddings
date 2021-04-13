#go build -o master main.go master.go monitor.go net.go util.go logs.go detector.go
#go build -o client main.go monitor.go net.go util.go logs.go detector.go client.go

go build -o main
# cd maple && go build -o ../wcmaple mapler.go wordcount.go && cd ..
# cd juice && go build -o ../wcjuice juicer.go wordcount.go

cd maple && go build -o ../condorcet_maple_1 mapler.go condorcet_1.go
cd ../maple && go build -o ../condorcet_maple_2 mapler.go condorcet_2.go

cd ../juice && go build -o ../condorcet_juice_1 juicer.go condorcet_1.go
cd ../juice && go build -o ../condorcet_juice_2 juicer.go condorcet_2.go

cd ../mj_wine && go build -o ../mapler mapler.go wine_maple.go
cd ../mj_wine && go build -o ../juicer juicer.go wine_juice.go

