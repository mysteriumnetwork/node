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
	testSuite.server = NewServer("localhost", 0, NewApiRouter())

	assert.NoError(testSuite.T(), testSuite.server.StartServing())
	port, err := testSuite.server.Port()
	assert.NoError(testSuite.T(), err)
	testSuite.client = NewTestClient(testSuite.T(), port)
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
