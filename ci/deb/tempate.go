package deb

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
)

const termsTemplate = `Template: mysterium/terms
Type: text
Description: You have to accept terms and conditions to install this software{{.}}

Template: mysterium/accept_terms
Type: boolean
Description: Do you accept Terms and Conditions?
 In order to install this package you have to accept its terms and conditions
`

func TermsTemplateFile(path string) error {
	templ := template.Must(template.New("terms").Parse(termsTemplate))

	resp, err := http.Get("https://raw.githubusercontent.com/mysteriumnetwork/terms/master/documents/TERMS_NODE_SHORT.md")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	terms, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	s := strings.ReplaceAll(string(terms), "\n", "\n ")

	err = templ.Execute(f, strings.ReplaceAll(s, " - ", " . "))
	if err != nil {
		return err
	}

	return nil
}
