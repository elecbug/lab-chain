sudo docker run \
    -dit \
    --name $1 \
    --network lab-chain-network \
    --env CONFIG_PATH=$2 \
    --mount type=bind,source=$(pwd)/config,target=/app/config \
    --mount type=bind,source=$(pwd)/data,target=/app/data \
    lab-chain-node 