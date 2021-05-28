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
	AddLabels        bool
	AddAnnotations   bool
	AddAll           bool
}

// EntitySubscriptions is a partial Entity definition for use with the
// PATCH /entities API
// type Deregistration struct {
//	Handler string `json:"handler"`
// }
type ObjectMeta struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// EntityPatch is a shell of an Entity object for use with the
// PATCH /entities API
type EntityPatch struct {
	Subscriptions []string   `json:"subscriptions,omitempty"`
	Metadata      ObjectMeta `json:"metadata,omitempty"`
	// TBD if we want to support other Entity-patchable fields:
	// CreatedBy        string            `json:"created_by,omitempty"`
	// EntityClass      string            `json:"entity_class,omitempty"`
	// User             string            `json:"user,omitempty"`
	// Deregister       string            `json:"deregister,omitempty"`
	// Deregistration   Deregistration    `json:"deregistration,omitempty"`
	// Redact           []string          `json:"redact"`
	// KeepaliveHandler string            `json:"keepalive_handler,omitempty"`
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
		{
			Path:      "api-url",
			Env:       "SENSU_API_URL",
			Argument:  "api-url",
			Shorthand: "a",
			Default:   "http://127.0.0.1:8080",
			Usage:     "Sensu API URL",
			Value:     &plugin.ApiUrl,
		},
		{
			Path:      "api-key",
			Env:       "SENSU_API_KEY",
			Argument:  "api-key",
			Shorthand: "k",
			Default:   "",
			Secret:    true,
			Usage:     "Sensu API Key",
			Value:     &plugin.ApiKey,
		},
		{
			Path:      "access-token",
			Env:       "SENSU_ACCESS_TOKEN",
			Argument:  "access-token",
			Shorthand: "t",
			Default:   "",
			Secret:    true,
			Usage:     "Sensu Access Token",
			Value:     &plugin.AccessToken,
		},
		{
			Path:      "trusted-ca-file",
			Env:       "SENSU_TRUSTED_CA_FILE",
			Argument:  "trusted-ca-file",
			Shorthand: "c",
			Default:   "",
			Usage:     "Sensu Trusted Certificate Authority file",
			Value:     &plugin.TrustedCaFile,
		},
		{
			Path:      "",
			Env:       "",
			Argument:  "add-subscriptions",
			Shorthand: "",
			Default:   false,
			Usage:     "Checks event.Check.Output for a newline-separated list of subscriptions to add",
			Value:     &plugin.AddSubscriptions,
		},
		{
			Path:      "",
			Env:       "",
			Argument:  "add-labels",
			Shorthand: "",
			Default:   false,
			Usage:     "Checks event.Check.Output for a newline-separated list of label key=value pairs to add",
			Value:     &plugin.AddLabels,
		},
		{
			Path:      "",
			Env:       "",
			Argument:  "add-annotations",
			Shorthand: "",
			Default:   false,
			Usage:     "Checks event.Check.Output for a newline-separated list of annotation key=value pairs to add",
			Value:     &plugin.AddAnnotations,
		},
		{
			Path:      "",
			Env:       "",
			Argument:  "add-all",
			Shorthand: "",
			Default:   false,
			Usage:     "Checks event.Check.Output for a newline-separated list of entity management commands to execute",
			Value:     &plugin.AddAll,
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
		fmt.Printf("Adding subscriptions from \"event.check.output\"\n")
		addSubscriptions(strings.Split(event.Check.Output, "\n"))
	}
	if len(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions"]) > 0 {
		fmt.Printf("Adding subscriptions from the \"sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions\" event annotation\n")
		addSubscriptions(strings.Split(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions"], ","))
	}
	if plugin.AddLabels {
		fmt.Printf("Adding labels from \"event.check.output\"\n")
		addLabels(strings.Split(event.Check.Output, "\n"))
	}
	if len(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/labels"]) > 0 {
		fmt.Printf("Adding labels from the \"sensu.io/plugins/sensu-entity-manager/config/patch/labels\" event annotation\n")
		addLabels(strings.Split(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/labels"], ","))
	}
	if plugin.AddAnnotations {
		fmt.Printf("Adding annotations from \"event.check.output\"\n")
		addAnnotations(strings.Split(event.Check.Output, "\n"))
	}
	if len(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/annotations"]) > 0 {
		fmt.Printf("Adding annotations from the \"sensu.io/plugins/sensu-entity-manager/config/patch/annotations\" event annotation\n")
		addAnnotations(strings.Split(event.Annotations["sensu.io/plugins/sensu-entity-manager/config/patch/annotations"], ","))
	}
	if plugin.AddAll {
		fmt.Printf("Adding entity properties from \"event.check.output\"\n")
		parseCommands(strings.Split(event.Check.Output, "\n"))
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

// Return the index location of a string in a []string
func indexOf(s []string, k string) int {
	for i, v := range s {
		if v == k {
			return i
		}
	}
	return -1
}

// Merge two map[string]string objects
// NOTE: this is a potentially destructive method (values may be overwritten)
func mergeMapStringStrings(a map[string]string, b map[string]string) map[string]string {
	if a == nil {
		fmt.Printf("Error: no entity labels; %v", a)
	}
	for k, v := range b {
		a[k] = v
	}
	return a
}

func mergeStringSlices(a []string, b []string) []string {
	for _, v := range b {
		if indexOf(a, v) < 0 {
			a = append(a, v)
		}
	}
	return a
}

func trimSlice(s []string) []string {
	for _, v := range s {
		if len(strings.TrimSpace(v)) == 0 {
			i := indexOf(s, v)
			s = trimSlice(append(s[:i], s[i+1:]...))
		}
	}
	return s
}

// Parse a slice of strings containing key=value pairs
func parseKvStringSlice(s []string) map[string]string {
	var m = make(map[string]string)
	for _, kvString := range s {
		i := strings.Split(kvString, "=")
		if len(i) > 1 {
			k := strings.TrimSpace(i[0])
			v := strings.TrimSpace(i[1])
			if len(strings.Split(k, " ")) > 1 {
				fmt.Printf("WARNING: invalid key name: \"%s\" (did you mean to use --add-all?)\n", k)
			} else {
				m[k] = v
			}
		}
	}
	return m
}

func addSubscriptions(subs []string) {
	plugin.Subscriptions = mergeStringSlices(plugin.Subscriptions, subs)
}

func addLabels(labels []string) {
	plugin.Labels = parseKvStringSlice(labels)
}

func addAnnotations(annotations []string) {
	plugin.Annotations = parseKvStringSlice(annotations)
}

func patchEntity(event *types.Event) *EntityPatch {
	entity := new(EntityPatch)

	// Merge subscriptions
	entity.Subscriptions = trimSlice(mergeStringSlices(event.Entity.Subscriptions, plugin.Subscriptions))

	// Init Metadata
	entity.Metadata = ObjectMeta{}

	// Merge labels
	entity.Metadata.Labels = mergeMapStringStrings(event.Entity.Labels, plugin.Labels)

	// Merge annotations
	entity.Metadata.Annotations = mergeMapStringStrings(event.Entity.Annotations, plugin.Annotations)

	return entity
}

// Parse commands
func parseCommands(s []string) {
	for _, str := range s {
		instructions := strings.Split(str, " ")
		if len(instructions) < 2 {
			fmt.Printf("WARNING: invalid command: \"%s\"\n", str)
		} else {
			command := strings.TrimSpace(instructions[0])
			argument := strings.TrimSpace(instructions[1])
			switch command {
			case "add-subscription":
				addSubscriptions([]string{argument})
			case "add-label":
				addLabels([]string{argument})
			case "add-annotation":
				addAnnotations([]string{argument})
			default:
				fmt.Printf("WARNING: nothing to do for command: \"%v\" (argument: \"%s\").\n", command, argument)
			}
		}
	}
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
}
