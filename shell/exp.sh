rm -rf data/log.jsonl

sudo docker rm -f boot
sudo docker rm -f node1
sudo docker rm -f node2
sudo docker rm -f node3
sudo docker rm -f node4
sudo docker rm -f node5

./shell/build.sh

./shell/run-back.sh boot config/boot.yaml boot-key