package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/nats-io/stan.go"
)

type TestData struct {
	Data      map[string]string
	Filenames []string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	sc, err := stan.Connect(os.Getenv("CLUSTER_NAME"), os.Getenv("CLIENT_ID"))
	if err != nil {
		log.Fatalf("Failed to connect to the cluster: %v", err)
	}
	defer sc.Close()

	testData := loadTestData()

	channel := os.Getenv("CHANNEL_NAME")

	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	go func() {
		for {
			randomIndex := r.Intn(6)
			msg := testData.Data[testData.Filenames[randomIndex]]

			if strings.HasPrefix(testData.Filenames[randomIndex], "valid") {
				uuid := uuid.New().String()
				msg = addUUIDToMessage(msg, uuid)
			}

			err := sc.Publish(channel, []byte(msg))
			if err != nil {
				log.Printf("Error posting message: %v", err)
			} else {
				log.Print("Message published")
			}

			time.Sleep(time.Second * 5)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
}

func loadTestData() TestData {
	testData := TestData{
		Data:      make(map[string]string),
		Filenames: []string{},
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	files := []string{"valid1.json", "valid2.json", "valid1.json", "not_valid1.json", "valid1.json", "valid2.json"}

	for _, file := range files {
		filePath := filepath.Join(wd, file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Error reading file %s: %v", filePath, err)
		}

		testData.Data[file] = string(data)
		testData.Filenames = append(testData.Filenames, file)
	}

	return testData
}

func addUUIDToMessage(msg string, uuid string) string {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(msg), &jsonData); err != nil {
		log.Fatalf("Error decoding JSON from message: %v", err)
	}

	jsonData["order_uid"] = uuid

	modifiedData, err := json.Marshal(jsonData)
	if err != nil {
		log.Fatalf("Error encoding JSON for message: %v", err)
	}

	return string(modifiedData)
}
