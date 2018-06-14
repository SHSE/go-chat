#!/usr/bin/env bash
docker-compose down
docker-compose up --build bench
trap "docker-compose kill && docker-compose down" EXIT
