package storage_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMemoryStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Memory Storage Suite")
}
