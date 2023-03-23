package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

type MockDbAdapterIterator struct {
	numIter int
	maxIter int
	data    []string
}

func (i *MockDbAdapterIterator) Next(ctx context.Context) bool {
	i.numIter++
	return i.numIter < i.maxIter
}
func (i *MockDbAdapterIterator) Close(ctx context.Context) error {
	return nil
}
func (i *MockDbAdapterIterator) Decode(val interface{}) error {
	return json.Unmarshal([]byte(i.data[i.numIter]), &val)
}

// MockDbAdapter use for unit tests
type MockDbAdapter struct {
	PingError bool
}

func NewMockDbAdapter() *MockDbAdapter {
	return &MockDbAdapter{
		PingError: false,
	}
}

func (c *MockDbAdapter) EnablePingError() {
	c.PingError = true
}

func (c *MockDbAdapter) DisablePingError() {
	c.PingError = false
}

func (c *MockDbAdapter) Close() error {
	return nil
}
func (c *MockDbAdapter) Ping() error {
	if c.PingError {
		return errors.New("Mock Ping Error")
	}
	return nil
}
func (c *MockDbAdapter) PingOK() bool {
	return !c.PingError
}
func (c *MockDbAdapter) Collection(collectionName string, databaseName ...string) *mongo.Collection {
	return nil
}
func (c *MockDbAdapter) WaitUntilStarted() {}
func (c *MockDbAdapter) Start()            {}
