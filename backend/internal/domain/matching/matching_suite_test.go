package matching_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMatching(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "domain/matching suite")
}
