#!/bin/bash

# Configure bitnami repository (requires for production ready charts)
helm repo add bitnami https://charts.bitnami.com/bitnami

# Create the namespace
kubectl create namespace bathyscaphe

# Install Redis
helm install --namespace bathyscaphe redis -f deployments/k8s/helm/redis-values.yaml bitnami/redis

# Install RabbitMQ
# helm install --namespace bathyscaphe rabbitmq -f deployments/k8s/helm/rabbitmq-values.yaml bitnami/rabbitmq

# Install our resources
kubectl -n bathyscaphe apply -f deployments/k8s/torproxy.yaml
# kubectl -n bathyscaphe apply -f deployments/k8s/configapi.yaml
