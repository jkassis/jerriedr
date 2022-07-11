#!/usr/local/bin/fish
make
set REGISTRY_HOST (oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
docker login -u kubeadmin -p (oc whoami -t) $REGISTRY_HOST
docker tag jerriedr-builder:latest $REGISTRY_HOST/fg/jerriedr-builder:latest
docker push $REGISTRY_HOST/fg/jerriedr-builder:latest
