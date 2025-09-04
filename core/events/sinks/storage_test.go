package sinks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/client"
	"github.com/dtomasi/k1s/core/events"
	"github.com/dtomasi/k1s/core/events/sinks"
)

const defaultTestKey = "default/test-event"

func TestStorageSinks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Sinks Suite")
}

var _ = Describe("StorageSink", func() {
	var (
		storageSink events.EventSink
		mockClient  *mockClient
		testEvent   *corev1.Event
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = newMockClient()

		testEvent = &corev1.Event{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Event",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "ConfigMap",
				Name:      "test-config",
				Namespace: "default",
			},
			Reason:  "Created",
			Message: "Test event message",
			Type:    corev1.EventTypeNormal,
			Count:   1,
		}

		storageSink = sinks.NewStorageSink(sinks.StorageSinkOptions{
			Client:  mockClient,
			Context: ctx,
		})
	})

	Describe("Create", func() {
		It("should create a new event", func() {
			result, err := storageSink.Create(testEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			// Verify event was stored
			key := defaultTestKey
			storedEvent, exists := mockClient.objects[key]
			Expect(exists).To(BeTrue())
			Expect(storedEvent.(*corev1.Event).GetName()).To(Equal("test-event"))
			Expect(storedEvent.(*corev1.Event).GetNamespace()).To(Equal("default"))
		})

		It("should set default namespace if not provided", func() {
			testEvent.Namespace = ""

			result, err := storageSink.Create(testEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Namespace).To(Equal(metav1.NamespaceDefault))
		})

		It("should handle creation errors", func() {
			mockClient.createError = errors.New("storage error")

			result, err := storageSink.Create(testEvent)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to create event"))
		})

		It("should try update if event already exists", func() {
			// Pre-store an event
			key := defaultTestKey
			existingEvent := testEvent.DeepCopy()
			existingEvent.Count = 2
			mockClient.objects[key] = existingEvent
			mockClient.createError = apierrors.NewAlreadyExists(schema.GroupResource{Resource: "events"}, "test-event")

			result, err := storageSink.Create(testEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			// Should have aggregated the count
			Expect(result.Count).To(Equal(int32(3))) // 2 + 1
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			// Pre-store an existing event
			key := defaultTestKey
			existingEvent := testEvent.DeepCopy()
			existingEvent.Count = 2
			existingEvent.FirstTimestamp = metav1.Now()
			existingEvent.LastTimestamp = metav1.Now()
			mockClient.objects[key] = existingEvent
		})

		It("should update an existing event", func() {
			newEvent := testEvent.DeepCopy()
			newEvent.Message = "Updated message"
			newEvent.Count = 1

			result, err := storageSink.Update(newEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			// Should aggregate count and update message
			Expect(result.Count).To(Equal(int32(3))) // 2 + 1
			Expect(result.Message).To(Equal("Updated message"))
		})

		It("should create event if not found", func() {
			delete(mockClient.objects, defaultTestKey)
			mockClient.getError = apierrors.NewNotFound(schema.GroupResource{Resource: "events"}, "test-event")

			result, err := storageSink.Update(testEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			// Should have created new event
			key := defaultTestKey
			storedEvent, exists := mockClient.objects[key]
			Expect(exists).To(BeTrue())
			Expect(storedEvent.(*corev1.Event).GetName()).To(Equal("test-event"))
		})

		It("should handle update errors", func() {
			mockClient.updateError = errors.New("storage error")

			result, err := storageSink.Update(testEvent)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to update event"))
		})
	})

	Describe("Patch", func() {
		BeforeEach(func() {
			// Pre-store an existing event
			key := defaultTestKey
			mockClient.objects[key] = testEvent.DeepCopy()
		})

		It("should patch an existing event", func() {
			patchData := []byte(`{"message": "Patched message"}`)

			result, err := storageSink.Patch(testEvent, patchData)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
		})

		It("should handle patch errors", func() {
			mockClient.patchError = errors.New("storage error")
			patchData := []byte(`{"message": "Patched message"}`)

			result, err := storageSink.Patch(testEvent, patchData)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to patch event"))
		})

		It("should validate input parameters", func() {
			result, err := storageSink.Patch(nil, []byte(`{}`))
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(Equal("event cannot be nil"))

			result, err = storageSink.Patch(testEvent, nil)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(Equal("patch data cannot be nil"))
		})
	})

	Describe("Event Aggregation", func() {
		It("should properly aggregate event counts", func() {
			// Create initial event
			key := defaultTestKey
			existingEvent := testEvent.DeepCopy()
			existingEvent.Count = 5
			existingEvent.FirstTimestamp = metav1.NewTime(time.Now().Add(-1 * time.Hour))
			existingEvent.LastTimestamp = metav1.NewTime(time.Now().Add(-30 * time.Minute))
			mockClient.objects[key] = existingEvent

			// Update with new event
			newEvent := testEvent.DeepCopy()
			newEvent.Count = 2
			newEvent.FirstTimestamp = metav1.Now()
			newEvent.LastTimestamp = metav1.Now()

			result, err := storageSink.Update(newEvent)
			Expect(err).NotTo(HaveOccurred())

			// Count should be aggregated
			Expect(result.Count).To(Equal(int32(7))) // 5 + 2

			// FirstTimestamp should be preserved (it's earlier)
			Expect(result.FirstTimestamp).To(Equal(existingEvent.FirstTimestamp))

			// LastTimestamp should be updated to the newer one
			Expect(result.LastTimestamp).To(Equal(newEvent.LastTimestamp))
		})

		It("should update message if different", func() {
			key := defaultTestKey
			existingEvent := testEvent.DeepCopy()
			existingEvent.Message = "Original message"
			mockClient.objects[key] = existingEvent

			newEvent := testEvent.DeepCopy()
			newEvent.Message = "Updated message"

			result, err := storageSink.Update(newEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Message).To(Equal("Updated message"))
		})

		It("should update reason and type if different", func() {
			key := defaultTestKey
			existingEvent := testEvent.DeepCopy()
			existingEvent.Reason = "OriginalReason"
			existingEvent.Type = corev1.EventTypeNormal
			mockClient.objects[key] = existingEvent

			newEvent := testEvent.DeepCopy()
			newEvent.Reason = "UpdatedReason"
			newEvent.Type = corev1.EventTypeWarning

			result, err := storageSink.Update(newEvent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Reason).To(Equal("UpdatedReason"))
			Expect(result.Type).To(Equal(corev1.EventTypeWarning))
		})
	})

	Describe("Factory Functions", func() {
		It("should create storage sink from client", func() {
			sink := sinks.CreateStorageSinkFromClient(mockClient)
			Expect(sink).NotTo(BeNil())
		})

		It("should create storage sink with context", func() {
			type testKey struct{}
			customCtx := context.WithValue(context.Background(), testKey{}, "value")
			sink := sinks.CreateStorageSinkWithContext(mockClient, customCtx)
			Expect(sink).NotTo(BeNil())
		})

		It("should work with event storage factory", func() {
			factory := &sinks.DefaultEventStorageFactory{}
			sink := factory.CreateEventStorageSink(sinks.StorageSinkOptions{
				Client:  mockClient,
				Context: ctx,
			})
			Expect(sink).NotTo(BeNil())
		})
	})
})

// Mock client implementation for testing

func newMockClient() *mockClient {
	return &mockClient{
		objects: make(map[string]runtime.Object),
	}
}

type mockClient struct {
	objects     map[string]runtime.Object
	createError error
	getError    error
	updateError error
	patchError  error
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.getError != nil {
		return m.getError
	}

	objectKey := key.String()
	if storedObj, exists := m.objects[objectKey]; exists {
		// Copy the stored object to the provided object
		storedEvent := storedObj.(*corev1.Event)
		targetEvent := obj.(*corev1.Event)
		*targetEvent = *storedEvent
		return nil
	}

	return apierrors.NewNotFound(schema.GroupResource{Resource: "events"}, key.Name)
}

func (m *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (m *mockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if m.createError != nil {
		return m.createError
	}

	key := client.ObjectKeyFromObject(obj).String()
	m.objects[key] = obj.(*corev1.Event)
	return nil
}

func (m *mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	key := client.ObjectKeyFromObject(obj).String()
	delete(m.objects, key)
	return nil
}

func (m *mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if m.updateError != nil {
		return m.updateError
	}

	key := client.ObjectKeyFromObject(obj).String()
	m.objects[key] = obj.(*corev1.Event)
	return nil
}

func (m *mockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if m.patchError != nil {
		return m.patchError
	}

	// For testing purposes, just treat patch as update
	return m.Update(ctx, obj, nil)
}

func (m *mockClient) Status() client.StatusWriter {
	return &mockStatusWriter{client: m}
}

func (m *mockClient) Scheme() *runtime.Scheme {
	return runtime.NewScheme()
}

func (m *mockClient) RESTMapper() meta.RESTMapper {
	return nil
}

type mockStatusWriter struct {
	client *mockClient
}

func (m *mockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return m.client.Update(ctx, obj, opts...)
}

func (m *mockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return m.client.Patch(ctx, obj, patch, opts...)
}
