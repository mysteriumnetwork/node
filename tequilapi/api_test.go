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
	testSuite.server, err = NewServer("localhost", 0, NewApiRouter())
	if err != nil {
		testSuite.T().FailNow()
	}
	testSuite.server.StartServing()
	testSuite.client = NewTestClient(testSuite.T(), testSuite.server.Port())
}

func (testSuite *tequilapiTestSuite) TestHealthCheckReturnsExpectedResponse() {
	resp := testSuite.client.Get("/healthcheck")

	expectJsonStatus200(testSuite.T(), resp, 200)

	var jsonMap map[string]interface{}
	parseResponseAsJson(testSuite.T(), resp, &jsonMap)
	assert.NotEmpty(testSuite.T(), jsonMap["uptime"])
}

func (testSuite *tequilapiTestSuite) TearDownSuite() {
	testSuite.server.Stop()
	testSuite.server.Wait()
}

func TestTequilaApiTestSuite(t *testing.T) {
	suite.Run(t, new(tequilapiTestSuite))
}
