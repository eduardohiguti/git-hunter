package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Hunter versionamento de arquivos")
		return
	}

	switch args[0] {
	case "oi":
		fmt.Println("oi")
	default: 
		fmt.Println("Comando não reconhecido:", args[0])
	}
}