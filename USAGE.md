# mir
Exploring to create ready to go IoT platform for embeded engineers.


### protoc

```sh
# binary
protoc cmds/protoproxy/gen/todo.proto --descriptor_set_out=cmds/protoproxy/gen/bin/todo.bproto

# with code gen

protoc --go_out=. --go_opt=paths=source_relative     --go-grpc_out=. --go-grpc_opt=paths=source_relative     cmds/protoproxy/gen/todo.proto [A

```
