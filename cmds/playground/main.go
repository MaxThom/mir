package main

import (
	"errors"
	"fmt"

	epkgs "github.com/pkg/errors"
)

func main() {
	e1 := errors.New("error one")
	e2 := errors.New("error two")
	e3 := errors.New("error three")
	e123 := errors.Join(e1, e2, e3)
	fmt.Println("combined\n", e123)
	var eAs any
	if errors.As(e1, &eAs) {
		fmt.Println("As ", eAs)
	}
	// var eIs error
	if errors.Is(e123, e2) {
		fmt.Println("Is ", e2)
	}

	wrappedError := fmt.Errorf("I am a wrapper %w %w", e1, e2)
	fmt.Println(errors.Unwrap(e123))
	fmt.Println(wrappedError)
	fmt.Println(errors.Unwrap(wrappedError))
	errors.Join(epkgs.New("asd"))

	eWrap := epkgs.Wrap(e1, "wrapper")
	fmt.Println(eWrap)

	eMsg := epkgs.WithMessage(e2, "message around e2")
	fmt.Println(eMsg)
	TestFunc()
}

func TestFunc() {
	eStack := epkgs.WithStack(epkgs.New("error stack"))
	fmt.Println(eStack)
}
