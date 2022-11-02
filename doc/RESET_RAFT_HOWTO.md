# Reset the RAFT

## Take the service down

> Login to oc

```
> oc login
> oc project fg
```

> Edit the StatefulSet Definition

```
Navigate to StatefulSets > <your stateful set>
```

> Save and remove the redeployment trigger annotations

```
  annotations:
    image.openshift.io/triggers: >-
      [{"from":{"kind":"ImageStreamTag","name":"permie:latest"},"fieldPath":"spec.template.spec.containers[?(@.name==\"app\")].image"}]
```

***Note***: If you don't do this, k8s will stomp the following change. So just save this now.

> Save the Currently Running Image ID

```
It looks like `image-registry.openshift-image-registry.svc:5000/fg/permie@sha256:71443bf116b7108ac4bf27653bcf0c0a9680bc848ee6456faa3c73f7c4c559fb`
```

> Replace the Running Image with the Image of the jerriedr

```
image-registry.openshift-image-registry.svc:5000/fg/jerriedr@sha256:b2aa3a1c40889a69abc3964dc1f3fea20d061ff2f1af40c0fa512b302bb314b1
```

***Note***: This does two things...
# It stops the database, releases locks, and stops incoming traffic.
# It starts the containers again with the jerriedr image so that the dr binary is available on each machine with their respective stateful sets attached.k

This makes it possible to get an exclusive lock on the database to make modifications.

> Get the current RAFT index

This is stored in the database and is flushed to disk when the RAFT WAL syncs. This is the index of the next RAFT proposal the stateful service expects to process. On startup, it gets sent to the RAFT which replays that proposal and all that follow it in the WAL (write ahead log).

We read the current raft index before and after modification to ensure that we have successfuly updated it.

```
./doctor raftIndexGet -d /var/data/single/*/data -i 0
```

> Delete all the RAFT data

***Note***: The system is designed to recreate the RAFT as it comes up. After this step you should reduce the replicas to 0 and then startup one replica at a time.

```
> rm -Rf $SERVICE_DATA_PERM_DIR/*/raft
```

> Reset the RAFT index

```
./doctor raftIndexSet -d /var/data/single/permie-2-server-0/data
```

> Reread the RAFT index

```
./doctor raftIndexGet -d /var/data/single/*/data -i 0
```

***Note***: Make sure it changed.

> Do this for each node

> Set the num replicas to 0

This ensures that the raft restarts starting with node 0

> Restore the redeployment trigger annotations

This will automatically update the image to latest.

