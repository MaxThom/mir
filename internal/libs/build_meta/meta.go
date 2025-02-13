package build_meta

// Build meta are modified through the linker flags on build time
// go build -ldflags
// "-X 'github.com/maxthom/mir-go/internal/libs/build_meta.Version=0.0.1'
// -X 'github.com/maxthom/mir-go/internal/libs/build_meta.User=maxthom' -X
// 'github.com/maxthom/mir-go/internal/libs/build_meta.Time=2021-08-01T00:00:00Z'"
//
// To find the path of a variable we want to change on build time, use go tool nm
// 1. build the app binary with go build
// 2. go tool nm ./<bin> | grep <package>

var Version string = "0.0.0"
var User string
var Time string

func GetShortVersion() string {
	return Version
}

func GetLongVersion() string {
	return Version + "\n" + Time + "\n" + User
}
