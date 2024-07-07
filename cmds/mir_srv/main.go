package main

import "fmt"

// IDEA here would be a single binary that contains every
// module part of Mir to run properly. Could simply have a set
// of feature flags switch on or off the modules that are needed.
// Integrate the cli and tui as well. They are top level and a
// subcommand for server with the custom cli. We can create a
// struct and inline the mir cmd

func main() {
	fmt.Println("hello world!")
}
