package main

import (
  "fmt"
  "os"
  "runtime"
)

func main() {
  fmt.Println("USER =", os.Getenv("USER"))
  fmt.Println("CLI args:", os.Args[1:])
  fmt.Println("Go version:", runtime.Version())
}
