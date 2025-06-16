sudo docker rm -f boot
sudo docker rm -f node1
sudo docker rm -f node2
sudo docker rm -f node3
sudo docker rm -f node4
sudo docker rm -f node5

./shell/build.sh

./shell/run-back.sh boot config/boot.yaml boot-key
./shell/run-back.sh node1 config/full.yaml node1-key
./shell/run-back.sh node2 config/full.yaml node2-key
./shell/run-back.sh node3 config/full.yaml node3-key
./shell/run-back.sh node4 config/full.yaml node4-key
./shell/run-front.sh node5 config/full.yaml node5-key