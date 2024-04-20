# Architecture Design Record

## ADR-17, Mir Kubernetes Operator

An operator could be there to install a production instance of the Mir ecosystem. It could provide advance features for scaling and have a single interface to define the system. 

### TUI & CLI

There could be a set of commands or windows to operate the Mir instance in Kubernetes. Such as:
- create new instance
- list instances `--[all-namespace|<namespace>]`
- delete instances
- update instances
In the end, those are just manipulation the CR.
It uses current kube context and namespace.
### Scaling of Protoproxy in Kubernetes

Here, I think the best scenario is to have some user inputs on how to do the scaling. More precisely, at what granularity of scaling. We could use a similar nomenclature of natsio message and subscribe patterns with `*.>`, but on the proto message name and namespace. 

Actually, this could also be their subscribe pattern. Thus the proto message full name need to be included. And this is defined by the publisher does the IoT Device. Could be retrieve from the pb on the client sdk side. An alternative could be to have a packet level stream and one grpc home made router that can use that provided pattern in the CR yaml to route properly.

The scaling of the protoproxy itself would be automatic with cpu or ram by Kubernetes. We must think about unordered data entering the database. This is also why we let the user choose its level.




