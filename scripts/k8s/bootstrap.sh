#!/bin/bash

helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add elastic https://helm.elastic.co

kubectl create namespace bathyscaphe

helm install --namespace bathyscaphe redis -f deployments/k8s/helm/redis-values.yaml bitnami/redis
helm install --namespace bathyscaphe rabbitmq -f deployments/k8s/helm/rabbitmq-values.yaml bitnami/rabbitmq
helm install --namespace bathyscaphe elasticsearch elastic/elasticsearch
helm install --namespace bathyscaphe kibana elastic/kibana

# Install our resources
kubectl -n bathyscaphe apply -f deployments/k8s/torproxy.yaml
kubectl -n bathyscaphe apply -f deployments/k8s/configapi.yaml
kubectl -n bathyscaphe apply -f deployments/k8s/crawler.yaml
kubectl -n bathyscaphe apply -f deployments/k8s/scheduler.yaml
kubectl -n bathyscaphe apply -f deployments/k8s/blacklister.yaml
kubectl -n bathyscaphe apply -f deployments/k8s/indexer-es.yaml
