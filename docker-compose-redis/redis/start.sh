#!/bin/bash
echo "begin"
redis-cli -a gf123456 --cluster add-node 192.168.3.28:7006  192.168.3.28:7000
echo "add-node"
sleep 5
redis-cli -a gf123456 --cluster rebalance 192.168.3.28:7006  --cluster-use-empty-masters