package hangul_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHangul(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "domain/hangul suite")
}
