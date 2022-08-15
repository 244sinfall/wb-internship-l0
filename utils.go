package main

import (
	"fmt"
	"github.com/bxcodec/faker"
)

func generateFakeMessage() Message {
	fakeMessage := Message{}
	err := faker.FakeData(&fakeMessage)
	if err != nil {
		fmt.Println("Error generating fake message: " + err.Error())
	}
	return fakeMessage
}
