package primitives

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

const runDir = "."

func TestGenerateRequiredFiles(t *testing.T) {
	sp, _ := GenerateOpenVPNSecPrimitives(runDir)

	if _, err := os.Stat(sp.CACertPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.CACertPath)
	}

	if _, err := os.Stat(sp.CAKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.CAKeyPath)
	}

	if _, err := os.Stat(sp.ServerCertPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.ServerCertPath)
	}

	if _, err := os.Stat(sp.ServerKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.ServerKeyPath)
	}

	if _, err := os.Stat(sp.TLSCryptKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.TLSCryptKeyPath)
	}
}

func TestDoubleGenerateFilesDiffer(t *testing.T) {
	sp, err := GenerateOpenVPNSecPrimitives(runDir)

	content1, err := ioutil.ReadFile(sp.ServerCertPath)
	if err != nil {
		t.Errorf("file %s should exist: %s", sp.ServerCertPath, err)
	}

	sp, err = GenerateOpenVPNSecPrimitives(runDir)

	content2, err := ioutil.ReadFile(sp.ServerCertPath)
	if err != nil {
		t.Errorf("file %s should exist: %s", sp.ServerCertPath, err)
	}

	assert.NotEqual(t, content1, content2)
}
