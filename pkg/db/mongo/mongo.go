package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoConfiguration struct {
	Host     string
	Port     string
	Username string
	Password string
}

func CreateMongoURI(config *MongoConfiguration) string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%s/", config.Username, config.Password, config.Host, config.Port)
}

type MongoService struct {
	client *mongo.Client
}

func NewMongoService(uri string) (*MongoService, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().SetTimeout(5 * time.Second).ApplyURI(uri).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	return &MongoService{
		client: client,
	}, nil
}

func (m *MongoService) Ping(ctx context.Context) error {
	err := m.client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoService) Disconnect(ctx context.Context) error {
	err := m.client.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (m *MongoService) GetCollection(db, coll string) *mongo.Collection {
	return m.client.Database(db).Collection(coll)
}

func (m *MongoService) FindOne(ctx context.Context, coll *mongo.Collection, filter any) (*mongo.SingleResult, error) {
	res := coll.FindOne(ctx, filter)
	if err := res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, svcErrors.ErrMongoNoDocs
		}
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}
	return res, nil
}

func (m *MongoService) InsertOne(ctx context.Context, coll *mongo.Collection, doc any) (*mongo.InsertOneResult, error) {
	res, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}
	return res, nil
}

func (m *MongoService) UpdateOne(ctx context.Context, coll *mongo.Collection, filter any, update any) (*mongo.UpdateResult, error) {
	res, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}
	if res.MatchedCount == 0 {
		return nil, svcErrors.ErrMongoNoDocs
	}
	return res, nil
}

func (m *MongoService) DeleteOne(ctx context.Context, coll *mongo.Collection, filter any) (*mongo.DeleteResult, error) {
	res, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}
	if res.DeletedCount == 0 {
		return nil, svcErrors.ErrMongoNoDocs
	}
	return res, nil
}

func (m *MongoService) Find(ctx context.Context, coll *mongo.Collection, filter any) (*mongo.Cursor, error) {
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}
	return cursor, nil
}

func (m *MongoService) DeleteMany(ctx context.Context, coll *mongo.Collection, filter any) (*mongo.DeleteResult, error) {
	res, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return nil, svcErrors.ErrMongo.WithInternalErr(err)
	}
	if res.DeletedCount == 0 {
		return nil, svcErrors.ErrMongoNoDocs
	}
	return res, nil
}
