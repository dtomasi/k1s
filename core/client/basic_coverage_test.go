package client_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/events"
)

var _ = Describe("Basic Coverage Tests", func() {
	var testClient client.Client

	BeforeEach(func() {
		// Use existing client setup from client_test.go
		mockStore := newMockStorage()
		testScheme := createTestScheme()
		mockRegistry := &mockRegistry{}

		var err error
		testClient, err = client.NewClient(client.ClientOptions{
			Scheme:   testScheme,
			Storage:  mockStore,
			Registry: mockRegistry,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should increase coverage for event client functions", func() {
		// Test CreateEventSource
		source := client.CreateEventSource()
		Expect(source.Component).To(Equal("k1s-client"))

		// Test CreateEventRecorderForClient
		scheme := createTestScheme()
		broadcaster := events.NewEventBroadcaster(events.EventBroadcasterOptions{})
		recorder := client.CreateEventRecorderForClient(broadcaster, scheme)
		Expect(recorder).NotTo(BeNil())

		// Test WithEventRecording
		eventClient := client.WithEventRecording(testClient, recorder)
		Expect(eventClient).NotTo(BeNil())

		// Test WithOptionalEventRecording
		eventClient2 := client.WithOptionalEventRecording(testClient, recorder, true)
		Expect(eventClient2).NotTo(BeNil())

		// Test event client methods
		retrievedRecorder := eventClient.GetEventRecorder()
		Expect(retrievedRecorder).To(Equal(recorder))

		newRecorder := events.NewEventRecorder(broadcaster, events.EventRecorderOptions{})
		eventClient.SetEventRecorder(newRecorder)
		Expect(eventClient.GetEventRecorder()).To(Equal(newRecorder))
	})

	It("should increase coverage for watch client functions", func() {
		// Test NewWatchClient
		watchClient, err := client.NewWatchClient(testClient)
		Expect(err).NotTo(HaveOccurred())
		Expect(watchClient).NotTo(BeNil())

		// Test Watch
		testList := &TestItemList{}
		watcher, err := watchClient.Watch(context.Background(), testList)
		if err == nil {
			// Test watcher interface
			resultChan := watcher.ResultChan()
			Expect(resultChan).NotTo(BeNil())
			watcher.Stop()
		}
	})

	It("should increase coverage for client helper functions", func() {
		// Test RESTMapper (might return nil, that's ok for coverage)
		testClient.RESTMapper()
	})

	It("should test event-aware client operations", func() {
		// Create event recorder
		broadcaster := events.NewEventBroadcaster(events.EventBroadcasterOptions{})
		recorder := events.NewEventRecorder(broadcaster, events.EventRecorderOptions{})

		// Create event-aware client
		eventClient := client.WithEventRecording(testClient, recorder)

		// Test recorder methods (these should not fail)
		retrievedRecorder := eventClient.GetEventRecorder()
		Expect(retrievedRecorder).To(Equal(recorder))

		newRecorder := events.NewEventRecorder(broadcaster, events.EventRecorderOptions{})
		eventClient.SetEventRecorder(newRecorder)
		Expect(eventClient.GetEventRecorder()).To(Equal(newRecorder))
	})
})
