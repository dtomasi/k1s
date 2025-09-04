package v1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV1Types(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Core Types v1 Suite")
}
