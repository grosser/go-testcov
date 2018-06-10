package go_scov

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAwesome(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Example")
}

var _ = Describe("Example", func() {
	Describe("Make an awesome expression", func() {
		Context("smile", func() {
			It("should result :)", func() {
				GetResult()
				Expect(":)").To(Equal(":)"))
			})
		})
	})
})
