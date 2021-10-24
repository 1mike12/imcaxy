package datahubstorage

import (
	"context"
	"errors"
)

type commonTopicRequest struct {
	topicID  string
	response chan error
}

func newCommonTopicRequest(topicID string) commonTopicRequest {
	return commonTopicRequest{
		topicID,
		make(chan error, 1),
	}
}

type closeTopicRequest struct {
	commonTopicRequest
	errorToForward error
}

func newCloseTopicRequest(topicID string, errorToFroward error) closeTopicRequest {
	return closeTopicRequest{
		newCommonTopicRequest(topicID),
		errorToFroward,
	}
}

type notification struct {
	closed bool
	err    error
}

type registerTopicListenerRequest struct {
	topicID  string
	response chan notification
}

func newRegisterTopicListenerRequest(topicID string) registerTopicListenerRequest {
	return registerTopicListenerRequest{
		topicID,
		make(chan notification, 1),
	}
}

type notificationHub struct {
	topics map[string][]chan notification

	registerTopic    chan commonTopicRequest
	closeTopic       chan closeTopicRequest
	sendNotification chan commonTopicRequest
	registerListener chan registerTopicListenerRequest
}

func newNotificationHub() notificationHub {
	return notificationHub{
		make(map[string][]chan notification),
		make(chan commonTopicRequest),
		make(chan closeTopicRequest),
		make(chan commonTopicRequest),
		make(chan registerTopicListenerRequest),
	}
}

func (hub *notificationHub) RegisterTopic(topicID string) <-chan error {
	request := newCommonTopicRequest(topicID)
	hub.registerTopic <- request
	return request.response
}

func (hub *notificationHub) CloseTopic(topicID string, errorToForward error) <-chan error {
	request := newCloseTopicRequest(topicID, errorToForward)
	hub.closeTopic <- request
	return request.response
}

func (hub *notificationHub) SendNotification(topicID string) <-chan error {
	request := newCommonTopicRequest(topicID)
	hub.sendNotification <- request
	return request.response
}

func (hub *notificationHub) OnNotify(topicID string) <-chan notification {
	request := newRegisterTopicListenerRequest(topicID)
	hub.registerListener <- request
	return request.response
}

func (hub *notificationHub) StartMonitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for topic, listeners := range hub.topics {
				for _, listener := range listeners {
					hub.sendNotificationOnChan(listener, true, context.Canceled)
				}
				delete(hub.topics, topic)
			}
			return

		case request := <-hub.registerTopic:
			_, exists := hub.topics[request.topicID]
			if exists {
				hub.sendResponseOnChan(request.response, errTopicAlreadyRegistered)
				continue
			}

			hub.topics[request.topicID] = make([]chan notification, 0)
			hub.sendResponseOnChan(request.response, nil)

		case request := <-hub.closeTopic:
			listeners, exists := hub.topics[request.topicID]
			if !exists {
				hub.sendResponseOnChan(request.response, errTopicNotFound)
				continue
			}

			for _, listener := range listeners {
				hub.sendNotificationOnChan(listener, true, request.errorToForward)
			}

			delete(hub.topics, request.topicID)
			hub.sendResponseOnChan(request.response, nil)

		case request := <-hub.sendNotification:
			listeners, exists := hub.topics[request.topicID]
			if !exists {
				hub.sendResponseOnChan(request.response, errTopicNotFound)
				continue
			}

			for _, listener := range listeners {
				hub.sendNotificationOnChan(listener, false, nil)
			}

			hub.topics[request.topicID] = make([]chan notification, 0)
			hub.sendResponseOnChan(request.response, nil)

		case request := <-hub.registerListener:
			listeners, exists := hub.topics[request.topicID]
			if !exists {
				hub.sendNotificationOnChan(request.response, true, errTopicNotFound)
				continue
			}

			hub.topics[request.topicID] = append(listeners, request.response)

			// this time we are not sending response, because request.response channel
			// is used only to notify about error or just to send nil notification
			// if we would use it now, it would be not reusable anymore
		}
	}
}

func (hub *notificationHub) sendNotificationOnChan(notificationChan chan notification, closed bool, errorToForward error) {
	n := notification{
		closed,
		errorToForward,
	}

	notificationChan <- n

	// we are closing particular notification channels
	// every time we send anything through it, because
	// listeners have to re-listen every time notification
	// was received
	close(notificationChan)
}

func (hub *notificationHub) sendResponseOnChan(responseChan chan error, err error) {
	responseChan <- err
	close(responseChan)
}

var (
	errTopicAlreadyRegistered = errors.New("topic already registered")
	errTopicNotFound          = errors.New("topic not found")
)
