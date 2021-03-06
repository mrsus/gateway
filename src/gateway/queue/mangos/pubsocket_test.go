package mangos_test

import (
	"fmt"
	"gateway/queue"
	qm "gateway/queue/mangos"
	"gateway/queue/testing"
	"reflect"
	"runtime"

	"github.com/go-mangos/mangos"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func (s *MangosSuite) TestPubSocket(c *gc.C) {
	m := &qm.PubSocket{}
	err := m.Bind("foo")
	c.Log("PubSocket can't Bind on nil socket")
	c.Check(err, gc.ErrorMatches, "mangos PubSocket couldn't Bind to foo: nil socket")

	c.Log("PubSocket Close with nil socket does nothing")
	err = m.Close() // Does nothing
	c.Check(err, jc.ErrorIsNil)

	p := getBasicPub(c, "tcp://localhost:9001")

	ch, e := p.Channels()
	c.Check(ch, gc.NotNil)
	c.Check(e, gc.NotNil)

	c.Log("live PubSocket Close does not error")
	c.Assert(p.Close(), jc.ErrorIsNil)
	_, ok := <-e
	c.Log("error channel should now be closed")
	c.Check(ok, gc.Equals, false)
}

func (s *MangosSuite) TestGetPubSocket(c *gc.C) {
	sc, err := qm.GetPubSocket(&qm.PubSocket{})
	c.Check(err, jc.ErrorIsNil)
	c.Check(sc, gc.IsNil)

	sc, err = qm.GetPubSocket(&testing.Publisher{})
	c.Check(err, gc.ErrorMatches, `getPubSocket expects \*mangos.PubSocket, got \*testing.Publisher`)

	p := getBasicPub(c, "tcp://localhost:9001")

	sc, err = qm.GetPubSocket(p)

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(sc, gc.NotNil)

	c.Assert(p.Close(), jc.ErrorIsNil)
}

func (s *MangosSuite) TestPubTCP(c *gc.C) {
	pTCP, err := queue.Publish(
		"tcp://localhost:9001",
		qm.Pub(false),
		qm.PubTCP,
	)

	c.Assert(err, jc.ErrorIsNil)
	c.Assert(pTCP, gc.NotNil)
	c.Assert(pTCP.Close(), jc.ErrorIsNil)

	_, err = qm.PubTCP(&qm.PubSocket{})
	c.Check(err, gc.ErrorMatches, "PubTCP requires a non-nil Socket, use Pub or XPub first")
}

func (s *MangosSuite) TestPubIPC(c *gc.C) {
	pIPC, err := queue.Publish(
		ipcTest,
		qm.Pub(false),
		qm.PubIPC,
	)

	switch runtime.GOOS {
	case "linux", "darwin":
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(pIPC, gc.NotNil)
		err = pIPC.Close()
		c.Assert(err, jc.ErrorIsNil)
	default:
		c.Check(err, gc.ErrorMatches, fmt.Sprintf("PubIPC failed: mangos IPC transport not supported on OS %q", runtime.GOOS))
		return // Don't need to test other behaviors
	}

	_, err = qm.PubIPC(&qm.PubSocket{})
	c.Check(err, gc.ErrorMatches, "PubIPC requires a non-nil Socket, use Pub or XPub first")
}

func (s *MangosSuite) TestPub(c *gc.C) {
	p, err := qm.Pub(false)(&testing.Publisher{})
	c.Assert(err, gc.ErrorMatches, `Pub expects nil Publisher, got \*testing.Publisher`)

	var qp queue.Publisher
	p, err = qm.Pub(false)(qp)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(p, gc.NotNil)
	c.Assert(reflect.TypeOf(p), gc.Equals, reflect.TypeOf(&qm.PubSocket{}))
	c.Assert(p.Close(), jc.ErrorIsNil)
}

func (s *MangosSuite) TestPubBufferSize(c *gc.C) {
	_, err := qm.PubBuffer(-10)(&qm.PubSocket{})
	c.Assert(err, gc.ErrorMatches, "PubBuffer expects positive size, got -10")
	_, err = qm.PubBuffer(10)(&testing.Publisher{})
	c.Assert(err, gc.ErrorMatches, `getPubSocket expects \*mangos.PubSocket, got \*testing.Publisher`)
	ps, err := qm.Pub(false)(queue.Publisher(nil))
	c.Assert(err, jc.ErrorIsNil)
	p, err := qm.PubBuffer(10)(ps)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(qm.GetPubBufferSize(p), gc.Equals, 10)

	p, err = queue.Publish(
		"tcp://localhost:9001",
		qm.Pub(false),
		qm.PubTCP,
		qm.PubBuffer(2048),
	)
	c.Assert(err, jc.ErrorIsNil)

	// Make sure it was set correctly on the socket itself
	sock, err := qm.GetPubSocket(p)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(s, gc.NotNil)

	buffSize, err := sock.GetOption(mangos.OptionWriteQLen)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(buffSize, gc.Equals, 2048)

	c.Assert(p.Close(), jc.ErrorIsNil)
}

func (s *MangosSuite) TestXPub(c *gc.C) {
	p, err := qm.XPub(&testing.Publisher{})
	c.Assert(err, gc.ErrorMatches, `XPub expects nil Publisher, got \*testing.Publisher`)

	c.Log("XPub makes a new socket")

	p, err = qm.XPub(queue.Publisher(nil))
	c.Assert(err, jc.ErrorIsNil)
	verifyXPub(c, p)

	c.Assert(p.Close(), jc.ErrorIsNil)

	c.Log("XPub works in Publish")
	p, err = queue.Publish(
		"tcp://localhost:9000",
		qm.XPub,
		qm.PubTCP,
	)
	c.Assert(err, jc.ErrorIsNil)
	verifyXPub(c, p)

	c.Assert(p.Close(), jc.ErrorIsNil)
}

func verifyXPub(c *gc.C, p queue.Publisher) {
	// Make sure it's non-nil
	c.Assert(p, gc.NotNil)
	c.Assert(reflect.TypeOf(p), gc.Equals, reflect.TypeOf(&qm.PubSocket{}))

	// Make sure the underlying socket was made
	xpSock, err := qm.GetPubSocket(p)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(xpSock, gc.NotNil)

	// Make sure the underlying socket was set to Raw mode
	isRawIf, err := xpSock.GetOption(mangos.OptionRaw)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(
		reflect.TypeOf(isRawIf),
		gc.Equals,
		reflect.TypeOf(interface{}(true)),
	)
	isRaw := isRawIf.(bool)
	c.Check(isRaw, gc.Equals, true)
}
