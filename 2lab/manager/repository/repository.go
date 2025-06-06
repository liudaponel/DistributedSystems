package repository

import (
	"context"
	"log"
	"manager/models"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName               = "cracker_db"
	collectionName             = "requests"
	pendingTasksCollectionName = "pending_tasks" // Коллекция для неотправленных задач
	dbOperationTimeout         = 10 * time.Second
)

type Repository struct {
	mongoClient           *mongo.Client
	collection            *mongo.Collection
	pendingTaskCollection *mongo.Collection
}

func NewRepository(mongoClient *mongo.Client) *Repository {
	return &Repository{
		mongoClient:           mongoClient,
		collection:            mongoClient.Database(databaseName).Collection(collectionName),
		pendingTaskCollection: mongoClient.Database(databaseName).Collection(pendingTasksCollectionName),
	}
}

func (r *Repository) SaveNewRequest(req models.StartRequest, numTaskParts int) models.RequestInfoMongo {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	requestID := uuid.NewString()
	requestInfo := models.RequestInfoMongo{
		ID:            requestID,
		Hash:          req.Hash,
		MaxLength:     req.MaxLength,
		Status:        models.StatusNew,
		TotalParts:    numTaskParts,
		ReceivedParts: 0,
		Data:          []string{},
	}

	r.collection.InsertOne(ctx, requestInfo)

	return requestInfo
}

func (r *Repository) UpdateTaskStatusTo(requestID string, newStatus models.RequestStatus, filter bson.M) *mongo.UpdateResult {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status": newStatus,
		},
	}
	result, _ := r.collection.UpdateOne(ctx, filter, update)
	return result
}

func (r *Repository) SavePendingTask(req models.WorkerRequest, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	task := models.PendingTask{
		ID:        uuid.NewString(),
		RequestID: req.RequestId,
		TaskBody:  body,
	}

	_, err := r.pendingTaskCollection.InsertOne(ctx, task)
	return err
}

func (r *Repository) AllPendingTasksCursor() (context.Context, context.CancelFunc, *mongo.Cursor) {
	findCtx, findCancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	cursor, _ := r.pendingTaskCollection.Find(findCtx, bson.M{})

	return findCtx, findCancel, cursor
}

func (r *Repository) DeletePendingTask(id string) {
	delCtx, delCancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	r.pendingTaskCollection.DeleteOne(delCtx, bson.M{"_id": id})
	delCancel()
}

func (r *Repository) FindRequest(requestID string) (models.RequestInfoMongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	filter := bson.M{"_id": requestID}
	var currentInfo models.RequestInfoMongo
	err := r.collection.FindOne(ctx, filter).Decode(&currentInfo)
	return currentInfo, err
}

func (r *Repository) UpdateTaskAfterReceivePart(res models.WorkerResponse, currentInfo models.RequestInfoMongo) {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout+5*time.Second)
	defer cancel()

	filter := bson.M{"_id": res.RequestId}
	update := bson.M{}
	setFields := bson.M{}
	pushFields := bson.M{}

	if len(res.Words) > 0 {
		pushFields["data"] = bson.M{"$each": res.Words}
	}
	countReceivedParts := currentInfo.ReceivedParts + 1
	if countReceivedParts >= currentInfo.TotalParts {
		setFields["status"] = models.StatusReady
		log.Printf("Request %s completed successfully! data: %v", res.RequestId, res.Words)
	} else {
		log.Printf("Received part %d/%d for request %s with data %v", countReceivedParts, currentInfo.TotalParts, res.RequestId, res.Words)
	}

	if len(setFields) > 0 {
		update["$set"] = setFields
	}
	update["$inc"] = bson.M{"received_parts": 1}
	if len(pushFields) > 0 {
		update["$push"] = pushFields
	}

	if len(update) > 0 {
		r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(false))
	}
}

func (r *Repository) CountConnections() int {
	ctx, cancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer cancel()

	filter := bson.M{"status": models.StatusInProgress}
	count, err := r.collection.CountDocuments(ctx, filter)

	if err != nil {
		return 0
	}
	return int(count)
}
