package datahubstorage

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/franela/goblin"
)

type feedbackChan struct {
	topic    string
	feedback <-chan notification
}

// Any code that calls SendNotification or CloseTopic,
// or cancells the context and tests responses on OnNotify
// should by encapsulated in testFunc function.
func catchNotificationsAfterTest(
	ctx context.Context,
	hub notificationHub,
	topicsToListen []string,
	testFunc func(topics []string),
) (responded map[string][]notification, notResponding []string) {
	feedbackChans := make([]feedbackChan, 0)
	for _, topic := range topicsToListen {
		feedbackCh := feedbackChan{topic, hub.OnNotify(topic)}
		feedbackChans = append(feedbackChans, feedbackCh)
	}

	testFunc(topicsToListen)

	responded = make(map[string][]notification)
	notResponding = make([]string, 0)

	// collect feedback from feedback channels
	for _, feedback := range feedbackChans {
		select {
		case <-time.After(time.Second):
			notResponding = append(notResponding, feedback.topic)
			continue

		case response := <-feedback.feedback:
			if _, exists := responded[feedback.topic]; !exists {
				responded[feedback.topic] = make([]notification, 0)
			}
			responded[feedback.topic] = append(responded[feedback.topic], response)
			continue

		case <-ctx.Done():
			return responded, notResponding
		}
	}

	return responded, notResponding
}

func catchNotifications(
	ctx context.Context,
	hub notificationHub,
	topicsToListen ...string,
) (responded map[string][]notification, notResponding []string) {
	return catchNotificationsAfterTest(ctx, hub, topicsToListen, func(_ []string) {})
}

func newRunningHub() (notificationHub, context.CancelFunc, context.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := newNotificationHub()

	go hub.StartMonitor(ctx)

	return hub, cancel, ctx
}

func newRunningHubWithMonitorStopSignal() (notificationHub, context.CancelFunc, <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := newNotificationHub()
	stopped := make(chan struct{}, 1)

	go func() {
		hub.StartMonitor(ctx)
		stopped <- struct{}{}
	}()

	return hub, cancel, stopped
}

func getErrorSafely(ctx context.Context, response <-chan error) error {
	timeout, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	select {
	case <-timeout.Done():
		return context.DeadlineExceeded
	case err := <-response:
		return err
	}
}

func TestNotificationHub(t *testing.T) {
	g := Goblin(t)

	g.Describe("notificationHub", func() {
		g.Describe("RegisterTopic", func() {
			g.It("Should register topic without error", func() {
				hub, cancel, ctx := newRunningHub()
				defer cancel()

				err := getErrorSafely(ctx, hub.RegisterTopic("test"))

				g.Assert(err).IsNil("expected to not get error, but got", err)
			})

			g.It("Should return error if topic is already registered", func() {
				hub, cancel, ctx := newRunningHub()
				defer cancel()

				hub.RegisterTopic("test")
				err := getErrorSafely(ctx, hub.RegisterTopic("test"))

				g.Assert(err).Eql(errTopicAlreadyRegistered, "expected to receive errTopicAlreadyRegistered error, but got", err)
			})
		})

		g.Describe("CloseTopic", func() {
			g.It("Should close topic without error", func() {
				hub, cancel, ctx := newRunningHub()
				defer cancel()

				hub.RegisterTopic("test")
				err := getErrorSafely(ctx, hub.CloseTopic("test", nil))

				g.Assert(err).IsNil("expected to execute without any error, but got", err)
			})

			g.It("Should return error if topic to close is not registered", func() {
				hub, cancel, ctx := newRunningHub()
				defer cancel()

				hub.RegisterTopic("test")
				err := getErrorSafely(ctx, hub.CloseTopic("unknown", nil))

				g.Assert(err).Equal(errTopicNotFound, "expected to receive errTopicNotFound error, but got", err)
			})
		})

		g.Describe("SendNotification", func() {
			g.It("Should return error if topic is not registered", func() {
				hub, cancel, ctx := newRunningHub()
				defer cancel()

				hub.RegisterTopic("test")
				err := getErrorSafely(ctx, hub.SendNotification("unknown"))

				g.Assert(err).Equal(errTopicNotFound, "expected to receive errTopicNotFound error, but got", err)
			})
		})

		g.Describe("OnNotify", func() {
			g.It("Should return error if topic is not registered", func() {
				hub, cancel, ctx := newRunningHub()
				defer cancel()

				hub.RegisterTopic("test")
				responded, _ := catchNotifications(ctx, hub, "unknown")
				result, exists := responded["unknown"]

				g.Assert(exists).IsTrue("topic unknown not responded")
				g.Assert(result[0].err).Equal(errTopicNotFound, "expected to receive errTopicNotFound error, but got", result[0].err)
			})
		})

		g.It("Should notify listener when notification is sent", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()

			hub.RegisterTopic("test")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test"},
				func(topics []string) {
					hub.SendNotification("test")
				},
			)
			response, exists := responded["test"]

			g.Assert(exists).IsTrue("topic test not responded")
			g.Assert(response[0].closed).IsFalse("topic test was closed")
			g.Assert(response[0].err).IsNil("topic test responded with error", response[0].err)
		})

		g.It("Should notify listener when topic is closed", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()

			hub.RegisterTopic("test")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test"},
				func(topics []string) {
					hub.CloseTopic("test", nil)
				},
			)
			response, exists := responded["test"]

			g.Assert(exists).IsTrue("topic test not responded")
			g.Assert(response[0].closed).IsTrue("topic test was not closed")
			g.Assert(response[0].err).IsNil("topic test responded with error", response[0].err)
		})

		g.It("Should notify listener when topic is closed with error", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()
			fetchingError := errors.New("fetching error")

			hub.RegisterTopic("test")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test"},
				func(topics []string) {
					hub.CloseTopic("test", fetchingError)
				},
			)
			response, exists := responded["test"]

			g.Assert(exists).IsTrue("topic test not responded")
			g.Assert(response[0].closed).IsTrue("topic test was not closed")
			g.Assert(response[0].err).Equal(fetchingError, "expected topic test to respond with fetching error, but got", response[0].err)
		})

		g.It("Should notify multiple listeners when notification is sent", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()

			hub.RegisterTopic("test")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test", "test"},
				func(topics []string) {
					hub.SendNotification("test")
				},
			)
			responses, exists := responded["test"]

			g.Assert(exists).IsTrue("topic test not responded")

			if len(responses) != 2 {
				g.Errorf("there should be recorded 2 responses, got %v", len(responses))
			}

			for index, response := range responses {
				g.Assert(response.closed).IsFalse("topic listener with index", index, "was closed")
				g.Assert(response.err).IsNil("topic listener with index", index, "returned error", response.err)
			}
		})

		g.It("Should notify multiple listeners when topic is closed", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()

			hub.RegisterTopic("test")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test", "test"},
				func(topics []string) {
					hub.CloseTopic("test", nil)
				},
			)
			responses, exists := responded["test"]

			g.Assert(exists).IsTrue("topic test not responded")

			if len(responses) != 2 {
				g.Errorf("there should be recorded 2 responses, got %v", len(responses))
			}

			for index, response := range responses {
				g.Assert(response.closed).IsTrue("topic listener with index", index, "was not closed")
				g.Assert(response.err).IsNil("topic listener with index", index, "returned error", response.err)
			}
		})

		g.It("Should notify multiple listeners when topic is closed with error", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()
			fetchingError := errors.New("fetching error")

			hub.RegisterTopic("test")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test", "test"},
				func(topics []string) {
					hub.CloseTopic("test", fetchingError)
				},
			)
			responses, exists := responded["test"]

			g.Assert(exists).IsTrue("topic test not responded")

			if len(responses) != 2 {
				g.Errorf("there should be recorded 2 responses, got %v", len(responses))
			}

			for index, response := range responses {
				g.Assert(response.closed).IsTrue("topic listener with index", index, "was not closed")
				g.Assert(response.err).Equal(fetchingError, "expected topic listener with index", index, "to return fetching error, but got", response.err)
			}
		})

		g.It("Should notify only correct listener when notification is sent", func() {
			hub, cancel, ctx := newRunningHub()
			defer cancel()

			hub.RegisterTopic("test")
			hub.RegisterTopic("other")
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				[]string{"test", "other"},
				func(topics []string) {
					hub.SendNotification("test")
					hub.CloseTopic("other", nil)
				},
			)

			responses, exists := responded["test"]
			g.Assert(exists).IsTrue("topic test not responded")
			g.Assert(responses[0].closed).IsFalse("topic test was closed, but should not be")
			g.Assert(responses[0].err).IsNil("got error on topic test", responses[0].err)

			responses, exists = responded["other"]
			g.Assert(exists).IsTrue("topic other not responded")
			g.Assert(responses[0].closed).IsTrue("topic other was not closed, but should be")
			g.Assert(responses[0].err).IsNil("got error on topic other", responses[0].err)

		})

		g.It("Should close all topics with context.Cancelled error when context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			hub, cancel, stopped := newRunningHubWithMonitorStopSignal()
			topics := []string{"first", "second", "third"}

			for _, topic := range topics {
				hub.RegisterTopic(topic)
			}
			responded, _ := catchNotificationsAfterTest(
				ctx,
				hub,
				topics,
				func(_ []string) {
					cancel()
					<-stopped
				},
			)

			for _, topic := range topics {
				result, exists := responded[topic]
				if !exists {
					g.Errorf("expected response on %s topic, but there was no response", topic)
				}

				g.Assert(len(result)).Equal(1, "result responses is not equal to 1")
				g.Assert(result[0].err).Equal(context.Canceled, "expected", topic, "topic to return context.Cancelled error, but got", result[0].err)
				g.Assert(result[0].closed).IsTrue("expected", topic, "topic to be closed, but topic was not closed")
			}
		})
	})
}
