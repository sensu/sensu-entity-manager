package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
)

// Config represents the handler plugin config.
type Config struct {
	sensu.PluginConfig
	AuthHeader       string
	ApiUrl           string
	ApiKey           string
	AccessToken      string
	TrustedCaFile    string
	Labels           map[string]string
	Annotations      map[string]string
	Subscriptions    []string
	AddSubscriptions bool
}

// EntitySubscriptions is a partial Entity definition for use with the
// PATCH /entities API
type Deregistration struct {
	Handler string `json:"handler"`
}

// EntityPatch is a shell of an Entity object for use with the
// PATCH /entities API
type EntityPatch struct {
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	CreatedBy        string            `json:"created_by,omitempty"`
	EntityClass      string            `json:"entity_class,omitempty"`
	User             string            `json:"user,omitempty"`
	Subscriptions    []string          `json:"subscriptions,omitempty"`
	Deregister       string            `json:"deregister,omitempty"`
	Deregistration   Deregistration    `json:"deregistration,omitempty"`
	Redact           []string          `json:"redact"`
	KeepaliveHandler string            `json:"keepalive_handler,omitempty"`
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-entity-manager",
			Short:    "Event-based Sensu entity management for service-discovery (add/remove subscriptions) and other automation workflows.",
			Keyspace: "sensu.io/plugins/sensu-entity-manager/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "api-url",
			Env:       "SENSU_API_URL",
			Argument:  "api-url",
			Shorthand: "a",
			Default:   "http://127.0.0.1:8080",
			Usage:     "Sensu API URL",
			Value:     &plugin.ApiUrl,
		},
		&sensu.PluginConfigOption{
			Path:      "api-key",
			Env:       "SENSU_API_KEY",
			Argument:  "api-key",
			Shorthand: "k",
			Default:   "",
			Secret:    true,
			Usage:     "Sensu API Key",
			Value:     &plugin.ApiKey,
		},
		&sensu.PluginConfigOption{
			Path:      "access-token",
			Env:       "SENSU_ACCESS_TOKEN",
			Argument:  "access-token",
			Shorthand: "t",
			Default:   "",
			Secret:    true,
			Usage:     "Sensu Access Token",
			Value:     &plugin.AccessToken,
		},
		&sensu.PluginConfigOption{
			Path:      "trusted-ca-file",
			Env:       "SENSU_TRUSTED_CA_FILE",
			Argument:  "trusted-ca-file",
			Shorthand: "c",
			Default:   "",
			Usage:     "Sensu Trusted Certificate Authority file",
			Value:     &plugin.TrustedCaFile,
		},
		&sensu.PluginConfigOption{
			Path:      "",
			Env:       "",
			Argument:  "add-subscriptions",
			Shorthand: "",
			Default:   false,
			Usage:     "Checks event.Check.Output for a newline-separated list of subscriptions to add",
			Value:     &plugin.AddSubscriptions,
		},
		&sensu.PluginConfigOption{
			Path:      "patch/subscriptions",
			Env:       "",
			Argument:  "",
			Shorthand: "",
			Default:   []string{},
			Usage:     "Sensu Entity Subscriptions",
			Value:     &plugin.Subscriptions,
		},
		&sensu.PluginConfigOption{
			Path:      "patch/labels",
			Env:       "",
			Argument:  "",
			Shorthand: "",
			Default:   "",
			Usage:     "Sensu Entity Labels",
			Value:     &plugin.Labels,
		},
		&sensu.PluginConfigOption{
			Path:      "patch/annotations",
			Env:       "",
			Argument:  "",
			Shorthand: "",
			Default:   "",
			Usage:     "Sensu Entity Annotations",
			Value:     &plugin.Annotations,
		},
	}
)

func main() {
	handler := sensu.NewGoHandler(&plugin.PluginConfig, options, checkArgs, executeHandler)
	handler.Execute()
}

func checkArgs(event *types.Event) error {
	if len(plugin.ApiKey) == 0 && len(plugin.AccessToken) == 0 {
		return fmt.Errorf("--api-key or $SENSU_API_KEY, or --access-token or $SENSU_ACCESS_TOKEN environment variable is required!")
	}
	if len(os.Getenv("SENSU_ACCESS_TOKEN")) > 0 {
		plugin.AccessToken = os.Getenv("SENSU_ACCESS_TOKEN")
		plugin.AuthHeader = fmt.Sprintf(
			"Bearer %s",
			os.Getenv("SENSU_API_KEY"),
		)
	}
	if len(os.Getenv("SENSU_API_KEY")) > 0 {
		plugin.ApiKey = os.Getenv("SENSU_API_KEY")
		plugin.AuthHeader = fmt.Sprintf(
			"Key %s",
			os.Getenv("SENSU_API_KEY"),
		)
	}
	if len(os.Getenv("SENSU_API_URL")) > 0 {
		plugin.ApiUrl = os.Getenv("SENSU_API_URL")
	}
	if plugin.AddSubscriptions {
		plugin.Subscriptions = strings.Split(event.Check.Output,"\n")
		fmt.Printf("Added %v subscriptions from event.Check.Output\n", len(plugin.Subscriptions))
	}
	if len(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions"]) > 0 {
		plugin.Subscriptions = strings.Split(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions"],",")
		fmt.Printf("Added %v subscriptions from the \"sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions\" event annotation\n", len(plugin.Subscriptions))
	}
	return nil
}

// LoadCACerts loads the system cert pool.
func LoadCACerts(path string) (*x509.CertPool, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		log.Printf("ERROR: failed to load system cert pool: %s", err)
		rootCAs = x509.NewCertPool()
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if path != "" {
		certs, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("ERROR: failed to read CA file (%s): %s", path, err)
			return nil, err
		}
		rootCAs.AppendCertsFromPEM(certs)
	}
	return rootCAs, nil
}

func initHTTPClient() *http.Client {
	certs, err := LoadCACerts(plugin.TrustedCaFile)
	if err != nil {
		log.Fatalf("ERROR: %s\n", err)
	}
	tlsConfig := &tls.Config{
		RootCAs: certs,
	}
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{
		Transport: tr,
	}
	return client
}

func indexOf(s []string, k string) int {
	for i, v := range s {
		if v == k {
			return i
		}
	}
	return -1
}

func mergeSlices(a []string, b []string) []string {
	for _, v := range b {
		if indexOf(a, v) < 0 {
			a = append(a, v)
		}
	}
	return a
}

func patchEntity(event *types.Event) *EntityPatch {
	entity := new(EntityPatch)

	// Merge subscriptions
	if len(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions"]) > 0 {
		plugin.Subscriptions = strings.Split(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions"], ",")
	}
	entity.Subscriptions = mergeSlices(event.Entity.Subscriptions, plugin.Subscriptions)

	return entity
}

func executeHandler(event *types.Event) error {
	data := patchEntity(event)
	postBody, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("ERROR: %s\n", err)
		return err
	}
	body := bytes.NewReader(postBody)
	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("%s/api/core/v2/namespaces/%s/entities/%s",
			plugin.ApiUrl,
			event.Entity.Namespace,
			event.Entity.Name,
		),
		body,
	)
	if err != nil {
		log.Fatalf("ERROR: %s\n", err)
	}
	var httpClient *http.Client = initHTTPClient()
	req.Header.Set("Authorization", plugin.AuthHeader)
	req.Header.Set("Content-Type", "application/merge-patch+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("ERROR: %s\n", err)
		return err
	} else if resp.StatusCode == 404 {
		log.Fatalf("ERROR: %v %s (%s)\n", resp.StatusCode, http.StatusText(resp.StatusCode), req.URL)
		return err
	} else if resp.StatusCode == 401 {
		log.Fatalf("ERROR: %v %s (%s)\n", resp.StatusCode, http.StatusText(resp.StatusCode), req.URL)
		return err
	} else if resp.StatusCode >= 300 {
		log.Fatalf("ERROR: %v %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		return err
	} else if resp.StatusCode == 204 {
		log.Printf("Successfully deleted entity \"%s\" from namespace \"%s\"", event.Entity.Name, event.Entity.Namespace)
		return nil
	} else {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("ERROR: %s\n", err)
			return err
		}
		fmt.Printf("%s\n", string(b))
		return nil
	}
	return nil
}
