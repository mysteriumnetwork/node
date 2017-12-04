package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type tequilapiTestSuite struct {
	suite.Suite
	server ApiServer
	client TestClient
}

func (testSuite *tequilapiTestSuite) SetupSuite() {
	var err error
	testSuite.server, err = Bootstrap("", 0)
	if err != nil {
		testSuite.T().FailNow()
	}
	testSuite.client = NewTestClient(testSuite.T(), testSuite.server.Port())
}

func (testSuite *tequilapiTestSuite) TestHealthCheckReturnsExpectedResponse() {

	resp := testSuite.client.Get("/healthcheck")

	var jsonMap map[string]interface{}
	expectJsonAndStatus(testSuite.T(), resp, 200, &jsonMap)

	assert.NotEmpty(testSuite.T(), jsonMap["uptime"])
}

func (testSuite *tequilapiTestSuite) TearDownSuite() {
	testSuite.server.Stop()
	testSuite.server.Wait()
}

func TestTequilaApiTestSuite(t *testing.T) {
	suite.Run(t, new(tequilapiTestSuite))
}
