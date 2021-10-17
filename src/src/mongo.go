package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type resource struct {
	Params   map[string]string `json:"params"`
	FilePath string            `json:"filePath"`
	FileID   string            `json:"fileID"`
}

type lastStartupTime struct {
	Date time.Time `json:"date"`
}

func connectToDB() (*mongo.Client, *context.Context, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://admin:admin@dev-imcaxy-mongo:27017"))
	if err != nil {
		return nil, nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}

	return client, &ctx, nil
}


func addStartupInfo(startupTime time.Time) {
	client, ctx, err := connectToDB()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(*ctx)

	collection := client.Database("statistics").Collection("startup_times")

	insertResult, err := collection.InsertOne(context.TODO(), lastStartupTime{Date: startupTime})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserted startup info with ID: ", insertResult.InsertedID)
}