sudo docker run \
    -dit \
    --name $1 \
    --network lab-chain-network \
    --env CONFIG_PATH=$2 \
    --mount type=bind,source=$(pwd)/config,target=/app/config \
    lab-chain-node 