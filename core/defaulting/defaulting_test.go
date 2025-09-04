package defaulting_test

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/defaulting"
)

// Test types for defaulting
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TestSpec   `json:"spec,omitempty"`
	Status            TestStatus `json:"status,omitempty"`
}

type TestSpec struct {
	Name        string   `json:"name"`
	Count       int32    `json:"count"`
	Price       *int64   `json:"price,omitempty"`
	Enabled     bool     `json:"enabled"`
	EnabledPtr  *bool    `json:"enabledPtr,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type TestStatus struct {
	Phase      string       `json:"phase"`
	Ready      bool         `json:"ready"`
	LastUpdate *metav1.Time `json:"lastUpdate,omitempty"`
}

// Implement runtime.Object interface
func (t *TestObject) DeepCopyObject() runtime.Object {
	if t == nil {
		return nil
	}
	out := new(TestObject)
	t.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another object of the same type
func (t *TestObject) DeepCopyInto(out *TestObject) {
	*out = *t
	t.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}

var _ = Describe("Defaulting Manager", func() {
	var (
		ctx     context.Context
		manager defaulting.DefaultingManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = defaulting.NewManager()
	})

	Describe("Manager Creation", func() {
		It("should create a new manager", func() {
			Expect(manager).NotTo(BeNil())
		})

		It("should not have defaults for unregistered types", func() {
			obj := &TestObject{}
			Expect(manager.HasDefaultsFor(obj)).To(BeFalse())
		})
	})

	Describe("Object Defaults Registration", func() {
		var objDefaults *defaulting.ObjectDefaults

		BeforeEach(func() {
			objDefaults = &defaulting.ObjectDefaults{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Defaults: []defaulting.DefaultValue{
					{
						FieldPath: "spec.count",
						Value:     int32(10),
						Type:      "int32",
					},
					{
						FieldPath: "spec.enabled",
						Value:     true,
						Type:      "bool",
					},
					{
						FieldPath: "status.phase",
						Value:     "Pending",
						Type:      "string",
					},
				},
			}
		})

		It("should register object defaults successfully", func() {
			err := manager.RegisterObjectDefaults(objDefaults)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject nil object defaults", func() {
			err := manager.RegisterObjectDefaults(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be nil"))
		})

		It("should reject object defaults with empty GVK", func() {
			objDefaults.GVK = schema.GroupVersionKind{}
			err := manager.RegisterObjectDefaults(objDefaults)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid GVK"))
		})
	})

	Describe("Strategy Registration", func() {
		var strategy defaulting.DefaultingStrategy

		BeforeEach(func() {
			strategy = defaulting.NewBasicStrategy(
				func(ctx context.Context, obj runtime.Object) error {
					if testObj, ok := obj.(*TestObject); ok {
						if testObj.Spec.Description == "" {
							testObj.Spec.Description = "Default description"
						}
					}
					return nil
				},
				reflect.TypeOf(&TestObject{}),
			)
		})

		It("should register a strategy successfully", func() {
			err := manager.RegisterStrategy(strategy)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject nil strategy", func() {
			err := manager.RegisterStrategy(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be nil"))
		})

		It("should unregister a strategy successfully", func() {
			err := manager.RegisterStrategy(strategy)
			Expect(err).NotTo(HaveOccurred())

			err = manager.UnregisterStrategy(strategy)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject nil strategy for unregistration", func() {
			err := manager.UnregisterStrategy(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be nil"))
		})
	})

	Describe("Default Application", func() {
		var (
			testObj     *TestObject
			objDefaults *defaulting.ObjectDefaults
		)

		BeforeEach(func() {
			testObj = &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-object",
				},
				Spec: TestSpec{
					Name: "Test Item",
				},
				Status: TestStatus{},
			}

			objDefaults = &defaulting.ObjectDefaults{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Defaults: []defaulting.DefaultValue{
					{
						FieldPath: "spec.count",
						Value:     "25",
						Type:      "int32",
					},
					{
						FieldPath: "spec.enabled",
						Value:     "true",
						Type:      "bool",
					},
					{
						FieldPath: "status.phase",
						Value:     "Initializing",
						Type:      "string",
					},
					{
						FieldPath: "status.ready",
						Value:     "false",
						Type:      "bool",
					},
				},
			}
		})

		Context("with object defaults", func() {
			BeforeEach(func() {
				err := manager.RegisterObjectDefaults(objDefaults)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should apply defaults to zero-value fields", func() {
				err := manager.Default(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())

				Expect(testObj.Spec.Count).To(Equal(int32(25)))
				Expect(testObj.Spec.Enabled).To(BeTrue())
				Expect(testObj.Status.Phase).To(Equal("Initializing"))
				Expect(testObj.Status.Ready).To(BeFalse())
			})

			It("should not override existing non-zero values", func() {
				testObj.Spec.Count = 100
				falseValue := false
				testObj.Spec.EnabledPtr = &falseValue
				testObj.Status.Phase = "Running"

				// Add a default for the pointer bool field
				objDefaults.Defaults = append(objDefaults.Defaults, defaulting.DefaultValue{
					FieldPath: "spec.enabledPtr",
					Value:     "true",
					Type:      "*bool",
				})
				err := manager.RegisterObjectDefaults(objDefaults)
				Expect(err).NotTo(HaveOccurred())

				err = manager.Default(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())

				Expect(testObj.Spec.Count).To(Equal(int32(100)))
				Expect(testObj.Spec.EnabledPtr).NotTo(BeNil())
				Expect(*testObj.Spec.EnabledPtr).To(BeFalse()) // Should not override existing pointer value
				Expect(testObj.Status.Phase).To(Equal("Running"))
				Expect(testObj.Status.Ready).To(BeFalse()) // Should still apply default for unset field
			})

			It("should handle pointer fields", func() {
				// Add pointer field default
				objDefaults.Defaults = append(objDefaults.Defaults, defaulting.DefaultValue{
					FieldPath: "spec.price",
					Value:     "1000",
					Type:      "*int64",
				})

				err := manager.RegisterObjectDefaults(objDefaults)
				Expect(err).NotTo(HaveOccurred())

				err = manager.Default(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())

				Expect(testObj.Spec.Price).NotTo(BeNil())
				Expect(*testObj.Spec.Price).To(Equal(int64(1000)))
			})

			It("should report having defaults for registered types", func() {
				Expect(manager.HasDefaultsFor(testObj)).To(BeTrue())
			})

			It("should handle nested struct initialization", func() {
				// Create object with nil nested struct
				testObjWithNilSpec := &TestObject{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "test.k1s.io/v1",
						Kind:       "TestObject",
					},
				}

				err := manager.Default(ctx, testObjWithNilSpec)
				Expect(err).NotTo(HaveOccurred())

				// Should have applied defaults to the spec fields
				Expect(testObjWithNilSpec.Spec.Count).To(Equal(int32(25)))
				Expect(testObjWithNilSpec.Spec.Enabled).To(BeTrue())
			})
		})

		Context("with strategy-based defaults", func() {
			var strategy defaulting.DefaultingStrategy

			BeforeEach(func() {
				strategy = defaulting.NewBasicStrategy(
					func(ctx context.Context, obj runtime.Object) error {
						if testObj, ok := obj.(*TestObject); ok {
							if testObj.Spec.Description == "" {
								testObj.Spec.Description = "Applied by strategy"
							}
							if testObj.Spec.Count == 0 {
								testObj.Spec.Count = 50
							}
						}
						return nil
					},
					reflect.TypeOf(&TestObject{}),
				)

				err := manager.RegisterStrategy(strategy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should apply strategy defaults", func() {
				err := manager.Default(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())

				Expect(testObj.Spec.Description).To(Equal("Applied by strategy"))
				Expect(testObj.Spec.Count).To(Equal(int32(50)))
			})

			It("should apply both strategy and object defaults", func() {
				err := manager.RegisterObjectDefaults(objDefaults)
				Expect(err).NotTo(HaveOccurred())

				err = manager.Default(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())

				// Strategy defaults applied first
				Expect(testObj.Spec.Description).To(Equal("Applied by strategy"))
				Expect(testObj.Spec.Count).To(Equal(int32(50)))

				// Object defaults applied after (but don't override strategy defaults)
				Expect(testObj.Spec.Enabled).To(BeTrue())
				Expect(testObj.Status.Phase).To(Equal("Initializing"))
			})

			It("should report having defaults for supported types", func() {
				Expect(manager.HasDefaultsFor(testObj)).To(BeTrue())
			})
		})

		Context("error handling", func() {
			It("should handle nil object", func() {
				err := manager.Default(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("nil object"))
			})

			It("should handle invalid field paths", func() {
				objDefaults.Defaults = []defaulting.DefaultValue{
					{
						FieldPath: "spec.nonexistentfield",
						Value:     "test",
						Type:      "string",
					},
				}

				err := manager.RegisterObjectDefaults(objDefaults)
				Expect(err).NotTo(HaveOccurred())

				err = manager.Default(ctx, testObj)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})

			It("should handle type conversion errors", func() {
				objDefaults.Defaults = []defaulting.DefaultValue{
					{
						FieldPath: "spec.count",
						Value:     "not-a-number",
						Type:      "int32",
					},
				}

				err := manager.RegisterObjectDefaults(objDefaults)
				Expect(err).NotTo(HaveOccurred())

				err = manager.Default(ctx, testObj)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("convert"))
			})
		})
	})

	Describe("Field Defaulter", func() {
		var (
			fieldDefaulter defaulting.FieldDefaulter
			testObj        *TestObject
		)

		BeforeEach(func() {
			fieldDefaulter = defaulting.NewFieldDefaulter(manager)
			testObj = &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
				Spec: TestSpec{
					Name: "Test Item",
				},
			}
		})

		It("should apply field default", func() {
			err := fieldDefaulter.DefaultField(ctx, testObj, "spec.count", int32(42))
			Expect(err).NotTo(HaveOccurred())
			Expect(testObj.Spec.Count).To(Equal(int32(42)))
		})

		It("should not override existing value", func() {
			testObj.Spec.Count = 99
			err := fieldDefaulter.DefaultField(ctx, testObj, "spec.count", int32(42))
			Expect(err).NotTo(HaveOccurred())
			Expect(testObj.Spec.Count).To(Equal(int32(99)))
		})

		It("should determine if field should be defaulted", func() {
			Expect(fieldDefaulter.ShouldDefault(ctx, testObj, "spec.count")).To(BeTrue())
			Expect(fieldDefaulter.ShouldDefault(ctx, testObj, "spec.name")).To(BeFalse())
		})

		It("should handle invalid field paths", func() {
			err := fieldDefaulter.DefaultField(ctx, testObj, "spec.invalid", "value")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Type Conversion", func() {
		var (
			testObj     *TestObject
			objDefaults *defaulting.ObjectDefaults
		)

		BeforeEach(func() {
			testObj = &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
			}
		})

		It("should convert string to int32", func() {
			objDefaults = &defaulting.ObjectDefaults{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Defaults: []defaulting.DefaultValue{
					{
						FieldPath: "spec.count",
						Value:     "123",
						Type:      "int32",
					},
				},
			}

			err := manager.RegisterObjectDefaults(objDefaults)
			Expect(err).NotTo(HaveOccurred())

			err = manager.Default(ctx, testObj)
			Expect(err).NotTo(HaveOccurred())

			Expect(testObj.Spec.Count).To(Equal(int32(123)))
		})

		It("should convert string to bool", func() {
			objDefaults = &defaulting.ObjectDefaults{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Defaults: []defaulting.DefaultValue{
					{
						FieldPath: "spec.enabled",
						Value:     "true",
						Type:      "bool",
					},
				},
			}

			err := manager.RegisterObjectDefaults(objDefaults)
			Expect(err).NotTo(HaveOccurred())

			err = manager.Default(ctx, testObj)
			Expect(err).NotTo(HaveOccurred())

			Expect(testObj.Spec.Enabled).To(BeTrue())
		})

		It("should handle direct value assignment", func() {
			objDefaults = &defaulting.ObjectDefaults{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Defaults: []defaulting.DefaultValue{
					{
						FieldPath: "spec.count",
						Value:     int32(456),
						Type:      "int32",
					},
				},
			}

			err := manager.RegisterObjectDefaults(objDefaults)
			Expect(err).NotTo(HaveOccurred())

			err = manager.Default(ctx, testObj)
			Expect(err).NotTo(HaveOccurred())

			Expect(testObj.Spec.Count).To(Equal(int32(456)))
		})
	})

	Describe("Concurrent Access", func() {
		It("should handle concurrent registration and defaulting", func() {
			done := make(chan bool)

			// Concurrent strategy registration
			go func() {
				defer GinkgoRecover()
				for i := 0; i < 10; i++ {
					strategy := defaulting.NewBasicStrategy(func(ctx context.Context, obj runtime.Object) error {
						return nil
					})
					err := manager.RegisterStrategy(strategy)
					Expect(err).NotTo(HaveOccurred())
				}
				done <- true
			}()

			// Concurrent defaulting
			go func() {
				defer GinkgoRecover()
				for i := 0; i < 10; i++ {
					testObj := &TestObject{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "test.k1s.io/v1",
							Kind:       "TestObject",
						},
					}
					_ = manager.Default(ctx, testObj)
				}
				done <- true
			}()

			// Wait for both goroutines to complete
			<-done
			<-done
		})
	})

	Describe("Edge Cases", func() {
		var testObj *TestObject

		BeforeEach(func() {
			testObj = &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
			}
		})

		It("should handle empty field path", func() {
			fieldDefaulter := defaulting.NewFieldDefaulter(manager)
			err := fieldDefaulter.DefaultField(ctx, testObj, "", "value")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty field path"))
		})

		It("should handle field paths with leading dots", func() {
			fieldDefaulter := defaulting.NewFieldDefaulter(manager)
			err := fieldDefaulter.DefaultField(ctx, testObj, ".spec.count", int32(100))
			Expect(err).NotTo(HaveOccurred())
			Expect(testObj.Spec.Count).To(Equal(int32(100)))
		})

		It("should handle objects without registered defaults", func() {
			// Create a test object without setting the GVK properly
			objWithoutGVK := &TestObject{
				Spec: TestSpec{Name: "test"},
			}
			// Don't set TypeMeta so GVK will be empty, but defaults will be inferred from type

			// This should not crash and should succeed (no defaults to apply)
			err := manager.Default(ctx, objWithoutGVK)
			Expect(err).NotTo(HaveOccurred())

			// Object should remain unchanged since no defaults are registered for this GVK
			Expect(objWithoutGVK.Spec.Name).To(Equal("test"))
			Expect(objWithoutGVK.Spec.Count).To(Equal(int32(0))) // Still zero value
		})
	})
})
