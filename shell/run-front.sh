sudo docker run \
    --rm \
    --name $1 \
    --network lab-chain-network \
    --mount type=bind,source=$(pwd)/config,target=/app/config \
    --mount type=bind,source=$(pwd)/data,target=/app/data \
    --env CONFIG_PATH=$2 \
    --env KEY_PATH=$3 \
    lab-chain-node 