package gomoljson

import (
	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type IssueSuite struct{}

func (s *IssueSuite) TestIssue4(t sweet.T) {
	cfg, _ := newFakeCfg()
	l, err := NewJSONLogger(cfg)
	Expect(err).To(BeNil())
	l.InitLogger()

	err = l.write([]byte{0x01, 0x02})
	Expect(err).To(BeNil())

	l.disconnect()

	err = l.write([]byte{0x01, 0x02})
	Expect(err).To(Equal(ErrDisconnected))
}
