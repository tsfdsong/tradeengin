#!/bin/bash

docker-compose -f deploy/dockers/docker-compose.yaml down

docker-compose -f deploy/dockerfiles/docker-compose.yaml down
