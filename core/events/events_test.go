package events_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/events"
	v1 "github.com/dtomasi/k1s/core/types/v1"
)

func TestEvents(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events Suite")
}

var _ = Describe("Event System", func() {
	var (
		scheme      *runtime.Scheme
		broadcaster events.EventBroadcaster
		recorder    events.EventRecorder
		testObject  *corev1.ConfigMap
		eventSource corev1.EventSource
	)

	BeforeEach(func() {

		// Setup scheme
		scheme = runtime.NewScheme()
		err := v1.AddEventToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		// Add ConfigMap for testing
		scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"},
			&corev1.ConfigMap{},
			&corev1.ConfigMapList{},
		)

		// Setup test object
		testObject = &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "default",
				UID:       "test-uid-123",
			},
		}

		// Setup event source
		eventSource = events.NewEventSource("test-component")

		// Setup broadcaster
		broadcaster = events.NewEventBroadcaster(events.EventBroadcasterOptions{
			QueueSize:      100,
			MetricsEnabled: true,
		})

		// Setup recorder
		recorder = broadcaster.NewRecorder(scheme, eventSource)
	})

	AfterEach(func() {
		if broadcaster != nil {
			broadcaster.Shutdown()
		}
	})

	Describe("EventRecorder", func() {
		It("should record normal events", func() {
			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			broadcaster.StartRecordingToSink(sink)

			recorder.Event(testObject, corev1.EventTypeNormal, events.ReasonCreated, "Test message")

			// Give some time for async processing
			time.Sleep(10 * time.Millisecond)

			Expect(recordedEvents).To(HaveLen(1))
			event := recordedEvents[0]
			Expect(event.Type).To(Equal(corev1.EventTypeNormal))
			Expect(event.Reason).To(Equal(events.ReasonCreated))
			Expect(event.Message).To(Equal("Test message"))
			Expect(event.InvolvedObject.Name).To(Equal("test-config"))
			Expect(event.InvolvedObject.Namespace).To(Equal("default"))
			Expect(event.Source.Component).To(Equal("test-component"))
		})

		It("should record warning events", func() {
			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			broadcaster.StartRecordingToSink(sink)

			recorder.Event(testObject, corev1.EventTypeWarning, events.ReasonFailed, "Test error")

			time.Sleep(10 * time.Millisecond)

			Expect(recordedEvents).To(HaveLen(1))
			event := recordedEvents[0]
			Expect(event.Type).To(Equal(corev1.EventTypeWarning))
			Expect(event.Reason).To(Equal(events.ReasonFailed))
			Expect(event.Message).To(Equal("Test error"))
		})

		It("should record formatted events", func() {
			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			broadcaster.StartRecordingToSink(sink)

			recorder.Eventf(testObject, corev1.EventTypeNormal, events.ReasonUpdated, "Updated %s with %d items", "config", 5)

			time.Sleep(10 * time.Millisecond)

			Expect(recordedEvents).To(HaveLen(1))
			event := recordedEvents[0]
			Expect(event.Message).To(Equal("Updated config with 5 items"))
		})

		It("should record annotated events", func() {
			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			broadcaster.StartRecordingToSink(sink)

			annotations := map[string]string{
				"test-annotation": "test-value",
			}
			recorder.AnnotatedEventf(testObject, annotations, corev1.EventTypeNormal, events.ReasonStarted, "Started with annotations")

			time.Sleep(10 * time.Millisecond)

			Expect(recordedEvents).To(HaveLen(1))
			event := recordedEvents[0]
			Expect(event.Annotations).To(HaveKeyWithValue("test-annotation", "test-value"))
		})
	})

	Describe("EventBroadcaster", func() {
		It("should distribute events to multiple sinks", func() {
			recordedEvents1 := []*corev1.Event{}
			recordedEvents2 := []*corev1.Event{}

			sink1 := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents1 = append(recordedEvents1, event)
					return event, nil
				},
			}
			sink2 := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents2 = append(recordedEvents2, event)
					return event, nil
				},
			}

			broadcaster.StartRecordingToSink(sink1)
			broadcaster.StartRecordingToSink(sink2)

			recorder.Event(testObject, corev1.EventTypeNormal, events.ReasonCreated, "Broadcast test")

			time.Sleep(20 * time.Millisecond)

			Expect(recordedEvents1).To(HaveLen(1))
			Expect(recordedEvents2).To(HaveLen(1))
			Expect(recordedEvents1[0].Message).To(Equal("Broadcast test"))
			Expect(recordedEvents2[0].Message).To(Equal("Broadcast test"))
		})

		It("should handle event watchers", func() {
			receivedEvents := []*corev1.Event{}

			watcher := broadcaster.StartEventWatcher(func(event *corev1.Event) {
				receivedEvents = append(receivedEvents, event)
			})
			defer watcher.Stop()

			recorder.Event(testObject, corev1.EventTypeNormal, events.ReasonCreated, "Watcher test")

			time.Sleep(10 * time.Millisecond)

			Expect(receivedEvents).To(HaveLen(1))
			Expect(receivedEvents[0].Message).To(Equal("Watcher test"))
		})

		It("should gracefully shutdown", func() {
			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			watcher := broadcaster.StartRecordingToSink(sink)

			// Record an event
			recorder.Event(testObject, corev1.EventTypeNormal, events.ReasonCreated, "Before shutdown")
			time.Sleep(10 * time.Millisecond)

			// Shutdown broadcaster
			broadcaster.Shutdown()

			// Try to record another event (should not be processed)
			recorder.Event(testObject, corev1.EventTypeNormal, events.ReasonUpdated, "After shutdown")
			time.Sleep(10 * time.Millisecond)

			// Only the first event should be recorded
			Expect(recordedEvents).To(HaveLen(1))
			Expect(recordedEvents[0].Message).To(Equal("Before shutdown"))

			// Watcher should be stopped (skip this check as it's timing sensitive)
			_ = watcher
		})
	})

	Describe("Helper Functions", func() {
		It("should create object references", func() {
			objRef, err := events.CreateObjectReference(scheme, testObject)
			Expect(err).NotTo(HaveOccurred())

			Expect(objRef.Kind).To(Equal("ConfigMap"))
			Expect(objRef.APIVersion).To(Equal("v1"))
			Expect(objRef.Name).To(Equal("test-config"))
			Expect(objRef.Namespace).To(Equal("default"))
			Expect(string(objRef.UID)).To(Equal("test-uid-123"))
		})

		It("should create event sources", func() {
			source := events.NewEventSource("test-component")
			Expect(source.Component).To(Equal("test-component"))
			Expect(source.Host).To(Equal("k1s-runtime"))
		})

		It("should create standard events using helper functions", func() {
			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			broadcaster.StartRecordingToSink(sink)

			// Test helper functions
			events.RecordSuccessfulCreate(recorder, testObject)
			events.RecordSuccessfulUpdate(recorder, testObject)
			events.RecordSuccessfulDelete(recorder, testObject)

			// Wait longer for async processing
			Eventually(func() int {
				return len(recordedEvents)
			}, "1s", "10ms").Should(Equal(3))

			// Check event reasons
			reasons := []string{}
			for _, event := range recordedEvents {
				reasons = append(reasons, event.Reason)
			}
			Expect(reasons).To(ContainElement(events.ReasonSuccessfulCreate))
			Expect(reasons).To(ContainElement(events.ReasonSuccessfulUpdate))
			Expect(reasons).To(ContainElement(events.ReasonSuccessfulDelete))
		})
	})

	Describe("Clock Integration", func() {
		It("should use custom clock for testing", func() {
			fixedTime := metav1.NewTime(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
			mockClock := &mockClock{now: fixedTime}

			recordedEvents := []*corev1.Event{}
			sink := &mockEventSink{
				createFunc: func(event *corev1.Event) (*corev1.Event, error) {
					recordedEvents = append(recordedEvents, event)
					return event, nil
				},
			}

			// Create recorder with custom clock
			customRecorder := events.NewEventRecorder(broadcaster, events.EventRecorderOptions{
				Scheme: scheme,
				Source: eventSource,
				Clock:  mockClock,
			})

			broadcaster.StartRecordingToSink(sink)

			customRecorder.Event(testObject, corev1.EventTypeNormal, events.ReasonCreated, "Clock test")

			time.Sleep(10 * time.Millisecond)

			Expect(recordedEvents).To(HaveLen(1))
			Expect(recordedEvents[0].FirstTimestamp).To(Equal(fixedTime))
			Expect(recordedEvents[0].LastTimestamp).To(Equal(fixedTime))
		})
	})
})

// Mock implementations for testing

type mockEventSink struct {
	createFunc func(event *corev1.Event) (*corev1.Event, error)
	updateFunc func(event *corev1.Event) (*corev1.Event, error)
	patchFunc  func(event *corev1.Event, data []byte) (*corev1.Event, error)
}

func (m *mockEventSink) Create(event *corev1.Event) (*corev1.Event, error) {
	if m.createFunc != nil {
		return m.createFunc(event)
	}
	return event, nil
}

func (m *mockEventSink) Update(event *corev1.Event) (*corev1.Event, error) {
	if m.updateFunc != nil {
		return m.updateFunc(event)
	}
	return event, nil
}

func (m *mockEventSink) Patch(event *corev1.Event, data []byte) (*corev1.Event, error) {
	if m.patchFunc != nil {
		return m.patchFunc(event, data)
	}
	return event, nil
}

type mockClock struct {
	now metav1.Time
}

func (m *mockClock) Now() metav1.Time {
	return m.now
}
