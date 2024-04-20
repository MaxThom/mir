# Architecture Design Record

## ADR-16, ProtoProxy


### Communication
- made to deserialise proto encoding from NatsIOs queue
- could be extented with a dynamic grpc library for direct comms

### Scaling of Protoproxy in Kubernetes

Using the help of an operator, we can scale
