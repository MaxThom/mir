# Goal: remove reflection

Using this [proto](https://github.com/aperturerobotics/protobuf-go-lite?tab=readme-ov-file) package

```yaml
# Replace plugins with this in buf.gen.yaml
plugins:
  - local: protoc-gen-go-lite
    out: proto/gen
    opt: paths=source_relative
```

Install those packages

```sh
go get github.com/aperturerobotics/protobuf-go-lite
```
