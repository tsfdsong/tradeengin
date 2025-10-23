#!/bin/bash

# 检查名为"engin"的网络是否存在
network_exists=$(docker network ls --format "{{.Name}}" --filter "name=engin")
# 如果网络不存在，则创建名为"gex"的网络
if [ -z "$network_exists" ]; then
    docker network create engin
    echo "网络 engin 创建成功！"
fi

docker-compose -f deploy/dockers/docker-compose.yaml up -d

host="127.0.0.1"
port="2379"

until nc -z $host $port; do
  echo "Etcd is unavailable - sleeping"
  sleep 2
done

docker-compose -f deploy/dockerfiles/docker-compose.yaml up -d



