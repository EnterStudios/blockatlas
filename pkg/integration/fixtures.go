// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"github.com/trustwallet/blockatlas/coin"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	fixturesFolder    = "testdata"            // Folder contains the JSON fixtures
	bodyFixturesFile  = "body_fixtures.json"  // Body fixtures for POST requests
	coinFixturesFile  = "coin_fixtures.json"  // Coin fixtures for path parameters
	queryFixturesFile = "query_fixtures.json" // Query string for GET requests
	excludeApisFile   = "exclude.json"        // API's need to be excluded from integration tests
)

type BodyFixture map[string]interface{}
type CoinFixture map[string]map[string]string
type QueryFixture map[string]map[string]interface{}
type ExcludeApis []string

var bodyFixture BodyFixture
var coinFixture CoinFixture
var queryFixture QueryFixture
var excludeApis ExcludeApis

func init() {
	logger.InitLogger()
	err := geFixtures(bodyFixturesFile, &bodyFixture)
	if err != nil {
		logger.Panic(err)
	}
	err = geFixtures(coinFixturesFile, &coinFixture)
	if err != nil {
		logger.Panic(err)
	}
	err = geFixtures(queryFixturesFile, &queryFixture)
	if err != nil {
		logger.Panic(err)
	}
	err = geFixtures(excludeApisFile, &excludeApis)
	if err != nil {
		logger.Panic(err)
	}
}

func geFixtures(f string, r interface{}) error {
	b, err := getFile(f)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b[:], &r)
	return err
}

func isExcluded(path string) bool {
	return contains(excludeApis, path)
}

func getFile(file string) ([]byte, error) {
	golden := filepath.Join(fixturesFolder, file)
	return ioutil.ReadFile(golden)
}

func getCoin(path string) coin.Coin {
	for _, c := range coin.Coins {
		if strings.Contains(path, fmt.Sprintf("/%s/", c.Handle)) {
			return c
		}
	}
	return coin.Coin{}
}

func getBody(path string) interface{} {
	fix, ok := bodyFixture[path]
	if !ok {
		return nil
	}
	return fix
}

func getQuery(path string) map[string]interface{} {
	fix, ok := queryFixture[path]
	if !ok {
		return nil
	}
	return fix
}

func addCoinFixtures(path string) string {
	c := getCoin(path)
	if (c == coin.Coin{}) {
		return path
	}
	fix, ok := coinFixture[c.Handle]
	if !ok {
		return strings.Replace(path, ":address", c.SampleAddr, -1)
	}
	if _, ok := fix["address"]; !ok {
		return strings.Replace(path, ":address", c.SampleAddr, -1)
	}
	result := path
	for key, value := range fix {
		result = strings.Replace(result, ":"+key, value, -1)
	}
	return result
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
