package main

import (
	"bufio"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"strings"
)

func main() {
	persistence := startSession()
	persistence.subscribe()
	fmt.Println("Now you can execute commands...")
	cmdReader := bufio.NewReader(os.Stdin)
	for {
		command, err := cmdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error while getting a command: " + err.Error())
			continue
		}
		trimmedCommand := strings.TrimSpace(command)
		err = persistence.executeCommand(trimmedCommand)
		if err != nil {
			fmt.Println("Error while executing a command: " + err.Error())
			continue
		}
	}
}
