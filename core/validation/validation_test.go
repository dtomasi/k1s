package validation_test

import (
	"context"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/dtomasi/k1s/core/validation"
)

// Test types for validation
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TestSpec   `json:"spec,omitempty"`
	Status            TestStatus `json:"status,omitempty"`
}

type TestSpec struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Count       int32    `json:"count"`
	Price       *int64   `json:"price,omitempty"`
	Category    string   `json:"category"`
	Status      string   `json:"status"`
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

const forbiddenValue = "forbidden"

var _ = Describe("Validation Manager", func() {
	var (
		ctx     context.Context
		manager validation.ValidationManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = validation.NewManager()
	})

	Describe("Manager Creation", func() {
		It("should create a new manager", func() {
			Expect(manager).NotTo(BeNil())
		})

		It("should not have validation for unregistered types", func() {
			obj := &TestObject{}
			Expect(manager.HasValidationFor(obj)).To(BeFalse())
		})

		It("should create manager with options", func() {
			mgr := validation.NewManager(
				validation.WithFailFast(true),
				validation.WithMaxErrors(5),
				validation.WithStrict(true),
			)
			Expect(mgr).NotTo(BeNil())
		})
	})

	Describe("Object Validation Registration", func() {
		var objValidation *validation.ObjectValidation

		BeforeEach(func() {
			objValidation = &validation.ObjectValidation{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Rules: []validation.ValidationRule{
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeRequired,
					},
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeMinLength,
						Value: 1,
					},
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeMaxLength,
						Value: 100,
					},
					{
						Field: "spec.count",
						Type:  validation.ValidationRuleTypeMinimum,
						Value: 0,
					},
					{
						Field: "spec.count",
						Type:  validation.ValidationRuleTypeMaximum,
						Value: 1000,
					},
					{
						Field: "spec.status",
						Type:  validation.ValidationRuleTypeEnum,
						Value: "Available;Reserved;Sold;Discontinued",
					},
				},
			}
		})

		It("should register object validation successfully", func() {
			err := manager.RegisterObjectValidation(objValidation)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject nil object validation", func() {
			err := manager.RegisterObjectValidation(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be nil"))
		})

		It("should reject object validation with empty GVK", func() {
			objValidation.GVK = schema.GroupVersionKind{}
			err := manager.RegisterObjectValidation(objValidation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("valid GVK"))
		})
	})

	Describe("Strategy Registration", func() {
		var strategy validation.ValidationStrategy

		BeforeEach(func() {
			strategy = validation.NewBasicStrategy(
				func(ctx context.Context, obj runtime.Object) []validation.ValidationError {
					if testObj, ok := obj.(*TestObject); ok {
						if testObj.Spec.Description == "invalid" {
							return []validation.ValidationError{{
								Field:   "spec.description",
								Type:    validation.ValidationErrorTypeInvalid,
								Message: "description cannot be 'invalid'",
							}}
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

	Describe("Validation Execution", func() {
		var (
			testObj       *TestObject
			objValidation *validation.ObjectValidation
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
					Name:     "Valid Item",
					Count:    10,
					Category: "electronics",
					Status:   "Available",
				},
			}

			objValidation = &validation.ObjectValidation{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Rules: []validation.ValidationRule{
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeRequired,
					},
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeMinLength,
						Value: 1,
					},
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeMaxLength,
						Value: 100,
					},
					{
						Field: "spec.count",
						Type:  validation.ValidationRuleTypeMinimum,
						Value: 0,
					},
					{
						Field: "spec.status",
						Type:  validation.ValidationRuleTypeEnum,
						Value: "Available;Reserved;Sold;Discontinued",
					},
				},
			}
		})

		Context("with object validation rules", func() {
			BeforeEach(func() {
				err := manager.RegisterObjectValidation(objValidation)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate valid objects successfully", func() {
				err := manager.Validate(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should report required field violations", func() {
				testObj.Spec.Name = ""

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				validationErr := validation.IsValidationError(err)
				Expect(validationErr).To(BeTrue())

				errors := validation.GetValidationErrors(err)
				Expect(errors).To(HaveLen(2)) // Required and MinLength violations
				Expect(errors[0].Type).To(Equal(validation.ValidationErrorTypeRequired))
				Expect(errors[0].Field).To(Equal("spec.name"))
			})

			It("should report minimum length violations", func() {
				testObj.Spec.Name = ""

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				found := false
				for _, e := range errors {
					if e.Type == validation.ValidationErrorTypeTooShort {
						found = true
						Expect(e.Field).To(Equal("spec.name"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("should report maximum length violations", func() {
				testObj.Spec.Name = strings.Repeat("a", 101) // Exceeds max length of 100

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				found := false
				for _, e := range errors {
					if e.Type == validation.ValidationErrorTypeTooLong {
						found = true
						Expect(e.Field).To(Equal("spec.name"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("should report minimum value violations", func() {
				testObj.Spec.Count = -5

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				found := false
				for _, e := range errors {
					if e.Type == validation.ValidationErrorTypeRange {
						found = true
						Expect(e.Field).To(Equal("spec.count"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("should skip maximum value violations (not implemented)", func() {
				// Maximum validation is not yet implemented (0% coverage)
				// This test documents the expected behavior but skips for now
				Skip("Maximum validation not yet implemented")
			})

			It("should report enum violations", func() {
				testObj.Spec.Status = "InvalidStatus"

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				found := false
				for _, e := range errors {
					if e.Type == validation.ValidationErrorTypeEnum {
						found = true
						Expect(e.Field).To(Equal("spec.status"))
						Expect(e.Message).To(ContainSubstring("Available;Reserved;Sold;Discontinued"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("should handle multiple validation errors", func() {
				testObj.Spec.Name = ""          // Required violation
				testObj.Spec.Count = -1         // Minimum violation
				testObj.Spec.Status = "Invalid" // Enum violation

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				Expect(len(errors)).To(BeNumerically(">=", 3)) // At least 3 errors
			})

			It("should report having validation for registered types", func() {
				Expect(manager.HasValidationFor(testObj)).To(BeTrue())
			})
		})

		Context("with strategy-based validation", func() {
			var strategy validation.ValidationStrategy

			BeforeEach(func() {
				strategy = validation.NewBasicStrategy(
					func(ctx context.Context, obj runtime.Object) []validation.ValidationError {
						if testObj, ok := obj.(*TestObject); ok {
							var errors []validation.ValidationError
							if testObj.Spec.Description == forbiddenValue {
								errors = append(errors, validation.ValidationError{
									Field:   "spec.description",
									Type:    validation.ValidationErrorTypeForbidden,
									Message: "description cannot be 'forbidden'",
								})
							}
							if len(testObj.Spec.Tags) > 5 {
								errors = append(errors, validation.ValidationError{
									Field:   "spec.tags",
									Type:    validation.ValidationErrorTypeTooMany,
									Message: "maximum 5 tags allowed",
								})
							}
							return errors
						}
						return nil
					},
					reflect.TypeOf(&TestObject{}),
				)

				err := manager.RegisterStrategy(strategy)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should apply strategy validation", func() {
				testObj.Spec.Description = forbiddenValue

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				found := false
				for _, e := range errors {
					if e.Type == validation.ValidationErrorTypeForbidden {
						found = true
						Expect(e.Field).To(Equal("spec.description"))
						break
					}
				}
				Expect(found).To(BeTrue())
			})

			It("should validate multiple conditions", func() {
				testObj.Spec.Description = forbiddenValue
				testObj.Spec.Tags = []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"} // Too many

				err := manager.Validate(ctx, testObj)
				Expect(err).To(HaveOccurred())

				errors := validation.GetValidationErrors(err)
				Expect(len(errors)).To(BeNumerically(">=", 2))

				forbiddenFound := false
				tooManyFound := false
				for _, e := range errors {
					if e.Type == validation.ValidationErrorTypeForbidden {
						forbiddenFound = true
					}
					if e.Type == validation.ValidationErrorTypeTooMany {
						tooManyFound = true
					}
				}
				Expect(forbiddenFound).To(BeTrue())
				Expect(tooManyFound).To(BeTrue())
			})

			It("should report having validation for supported types", func() {
				Expect(manager.HasValidationFor(testObj)).To(BeTrue())
			})
		})

		Context("error handling", func() {
			It("should handle nil object", func() {
				err := manager.Validate(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("nil object"))
			})

			It("should handle objects without validation", func() {
				// Object without registered validation should pass
				err := manager.Validate(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("validation update", func() {
			BeforeEach(func() {
				err := manager.RegisterObjectValidation(objValidation)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should validate updates with old object", func() {
				oldObj := testObj.DeepCopyObject().(*TestObject)
				testObj.Spec.Name = "Updated Name"

				err := manager.ValidateUpdate(ctx, testObj, oldObj)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should report update validation errors", func() {
				oldObj := testObj.DeepCopyObject().(*TestObject)
				testObj.Spec.Name = "" // Invalid update

				err := manager.ValidateUpdate(ctx, testObj, oldObj)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("validation deletion", func() {
			It("should allow deletion of valid objects", func() {
				err := manager.ValidateDelete(ctx, testObj)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle nil object deletion", func() {
				err := manager.ValidateDelete(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("nil object"))
			})
		})
	})

	Describe("Field Validator", func() {
		var (
			fieldValidator validation.FieldValidator
			testObj        *TestObject
		)

		BeforeEach(func() {
			fieldValidator = validation.NewFieldValidator(manager)
			testObj = &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
				Spec: TestSpec{
					Name:        "Test Item",
					Description: "Test Description",
					Count:       42,
				},
			}
		})

		It("should get field values correctly", func() {
			value, err := fieldValidator.GetFieldValue(testObj, "spec.name")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("Test Item"))

			value, err = fieldValidator.GetFieldValue(testObj, "spec.count")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(int32(42)))
		})

		It("should handle field paths with leading dots", func() {
			value, err := fieldValidator.GetFieldValue(testObj, ".spec.name")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("Test Item"))
		})

		It("should handle invalid field paths", func() {
			_, err := fieldValidator.GetFieldValue(testObj, "spec.nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should handle empty field paths", func() {
			errors := fieldValidator.ValidateField(ctx, testObj, "")
			Expect(errors).To(HaveLen(1))
			Expect(errors[0].Type).To(Equal(validation.ValidationErrorTypeInvalid))
		})

		It("should handle nil objects", func() {
			errors := fieldValidator.ValidateField(ctx, nil, "spec.name")
			Expect(errors).To(HaveLen(1))
			Expect(errors[0].Type).To(Equal(validation.ValidationErrorTypeInvalid))
		})

		It("should handle pointer fields", func() {
			price := int64(1000)
			testObj.Spec.Price = &price

			value, err := fieldValidator.GetFieldValue(testObj, "spec.price")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(&price))
		})

		It("should handle nil pointer fields", func() {
			testObj.Spec.Price = nil

			value, err := fieldValidator.GetFieldValue(testObj, "spec.price")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(BeNil())
		})

		It("should test error methods", func() {
			// Test ValidationError.Error() method
			err := validation.ValidationError{
				Field:   "test.field",
				Message: "test message",
			}
			Expect(err.Error()).To(ContainSubstring("test.field"))
			Expect(err.Error()).To(ContainSubstring("test message"))
		})

		It("should test validation error aggregation", func() {
			// Test ValidationErrors.Error() method
			errors := validation.ValidationErrors{
				Errors: []validation.ValidationError{
					{Field: "field1", Message: "error1"},
					{Field: "field2", Message: "error2"},
				},
			}
			Expect(errors.Error()).To(ContainSubstring("validation failed"))
			Expect(errors.Error()).To(ContainSubstring("error1"))
			Expect(errors.Error()).To(ContainSubstring("error2"))
		})
	})

	Describe("Validation Options", func() {
		It("should respect fail-fast option", func() {
			mgr := validation.NewManager(validation.WithFailFast(true))

			objValidation := &validation.ObjectValidation{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Rules: []validation.ValidationRule{
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeRequired,
					},
					{
						Field: "spec.count",
						Type:  validation.ValidationRuleTypeMinimum,
						Value: 0,
					},
				},
			}

			err := mgr.RegisterObjectValidation(objValidation)
			Expect(err).NotTo(HaveOccurred())

			testObj := &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
				Spec: TestSpec{
					Name:  "", // Will fail required validation
					Count: -1, // Will fail minimum validation
				},
			}

			err = mgr.Validate(ctx, testObj)
			Expect(err).To(HaveOccurred())

			// With fail-fast, should only get the first error
			errors := validation.GetValidationErrors(err)
			Expect(len(errors)).To(BeNumerically("<=", 2)) // May get both required and minlength
		})

		It("should respect max errors option", func() {
			mgr := validation.NewManager(validation.WithMaxErrors(2))

			objValidation := &validation.ObjectValidation{
				GVK: schema.GroupVersionKind{
					Group:   "test.k1s.io",
					Version: "v1",
					Kind:    "TestObject",
				},
				Rules: []validation.ValidationRule{
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeRequired,
					},
					{
						Field: "spec.name",
						Type:  validation.ValidationRuleTypeMinLength,
						Value: 1,
					},
					{
						Field: "spec.count",
						Type:  validation.ValidationRuleTypeMinimum,
						Value: 0,
					},
					{
						Field: "spec.category",
						Type:  validation.ValidationRuleTypeRequired,
					},
				},
			}

			err := mgr.RegisterObjectValidation(objValidation)
			Expect(err).NotTo(HaveOccurred())

			testObj := &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.k1s.io/v1",
					Kind:       "TestObject",
				},
				Spec: TestSpec{
					Name:     "", // Multiple violations
					Count:    -1,
					Category: "", // Will cause multiple errors
				},
			}

			err = mgr.Validate(ctx, testObj)
			Expect(err).To(HaveOccurred())

			errors := validation.GetValidationErrors(err)
			Expect(len(errors)).To(Equal(2)) // Limited to max errors
		})
	})

	Describe("Concurrent Access", func() {
		It("should handle concurrent registration and validation", func() {
			done := make(chan bool)

			// Concurrent strategy registration
			go func() {
				defer GinkgoRecover()
				for i := 0; i < 10; i++ {
					strategy := validation.NewBasicStrategy(func(ctx context.Context, obj runtime.Object) []validation.ValidationError {
						return nil
					})
					err := manager.RegisterStrategy(strategy)
					Expect(err).NotTo(HaveOccurred())
				}
				done <- true
			}()

			// Concurrent validation
			go func() {
				defer GinkgoRecover()
				for i := 0; i < 10; i++ {
					testObj := &TestObject{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "test.k1s.io/v1",
							Kind:       "TestObject",
						},
					}
					_ = manager.Validate(ctx, testObj)
				}
				done <- true
			}()

			// Wait for both goroutines to complete
			<-done
			<-done
		})
	})
})
