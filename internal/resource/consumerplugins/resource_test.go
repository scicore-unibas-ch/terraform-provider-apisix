package consumerplugins_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/acctest"
)

const testConsumer = "tf_acc_consumer_plugins_user"

// The credential this resource must never touch. The point of
// apisix_consumer_plugins is that some other system owns the consumer and its
// key, so the fixture is created over the Admin API rather than with
// apisix_consumer — using the latter would make Terraform own the object and
// the two resources would fight over the same plugins map.
const testKey = "tf-acc-consumer-plugins-key"

func adminRequest(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal %s %s: %v", method, path, err)
		}
	}

	base := strings.TrimRight(os.Getenv("APISIX_BASE_URL"), "/")
	req, err := http.NewRequest(method, base+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	req.Header.Set("X-API-KEY", os.Getenv("APISIX_ADMIN_KEY"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// createFixtureConsumer stands in for the external system (a portal, an IdP)
// that owns consumers. Registers its own cleanup.
func createFixtureConsumer(t *testing.T) {
	t.Helper()

	resp := adminRequest(t, http.MethodPut, "/consumers/"+testConsumer, map[string]any{
		"username": testConsumer,
		"plugins":  map[string]any{"key-auth": map[string]any{"key": testKey}},
		"labels":   map[string]string{"owner": "external-system"},
	})
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		t.Fatalf("create fixture consumer: HTTP %d", resp.StatusCode)
	}

	t.Cleanup(func() {
		r := adminRequest(t, http.MethodDelete, "/consumers/"+testConsumer, nil)
		r.Body.Close()
	})
}

// consumerPlugins returns the plugins map currently on the fixture consumer.
func consumerPlugins(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	resp := adminRequest(t, http.MethodGet, "/consumers/"+testConsumer, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read fixture consumer: HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Value struct {
			Plugins map[string]json.RawMessage `json:"plugins"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		t.Fatalf("decode fixture consumer: %v", err)
	}
	return wrapper.Value.Plugins
}

// checkKeyAuthIntact asserts the credential survived whatever Terraform just
// did. This is the guarantee the resource exists to provide.
func checkKeyAuthIntact(t *testing.T) resource.TestCheckFunc {
	return func(*terraform.State) error {
		plugins := consumerPlugins(t)
		raw, ok := plugins["key-auth"]
		if !ok {
			return fmt.Errorf("key-auth was removed from consumer %q", testConsumer)
		}
		var keyAuth struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(raw, &keyAuth); err != nil {
			return fmt.Errorf("decode key-auth: %w", err)
		}
		if keyAuth.Key != testKey {
			return fmt.Errorf("key-auth key changed: got %q, want %q", keyAuth.Key, testKey)
		}
		return nil
	}
}

func config(limit int) string {
	return fmt.Sprintf(`
resource "apisix_consumer_plugins" "quota" {
  consumer_id = %q

  plugins = {
    "limit-count" = jsonencode({
      count         = %d
      time_window   = 60
      rejected_code = 429
    })
  }
}
`, testConsumer, limit)
}

// TestAccConsumerPlugins_stateStability covers the standard guarantees plus
// the two specific to partial ownership:
//
//  1. Create attaches the plugin without disturbing the consumer's credential.
//  2. Update in place converges, credential still intact.
//  3. A no-op re-plan is empty.
//  4. Import round-trips (ID is "<consumer>/<plugin>").
//  5. Destroy detaches only the managed plugin — the consumer and its
//     key-auth credential survive.
func TestAccConsumerPlugins_stateStability(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t); createFixtureConsumer(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			plugins := consumerPlugins(t)
			if _, still := plugins["limit-count"]; still {
				return fmt.Errorf("limit-count still attached after destroy")
			}
			if _, ok := plugins["key-auth"]; !ok {
				return fmt.Errorf("destroy removed key-auth; it must survive")
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: config(10000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"apisix_consumer_plugins.quota", "consumer_id", testConsumer,
					),
					resource.TestCheckResourceAttrSet(
						"apisix_consumer_plugins.quota", "plugins.limit-count",
					),
					checkKeyAuthIntact(t),
				),
			},
			{
				Config: config(20000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(
						"apisix_consumer_plugins.quota", "plugins.limit-count",
						regexp.MustCompile("20000"),
					),
					checkKeyAuthIntact(t),
				),
			},
			{
				Config:             config(20000),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:  "apisix_consumer_plugins.quota",
				ImportState:   true,
				ImportStateId: testConsumer + "/limit-count",
				// This resource has no "id": it does not own an APISIX object,
				// it decorates one owned elsewhere. consumer_id is the
				// identifier the framework should match on.
				ImportStateVerifyIdentifierAttribute: "consumer_id",
				ImportStateVerify:                    true,
			},
		},
	})
}

// TestAccConsumerPlugins_importRequiresPluginNames asserts that importing by
// bare consumer name is refused: it would otherwise pull key-auth into state
// and the next apply would take ownership of the credential.
func TestAccConsumerPlugins_importRequiresPluginNames(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t); createFixtureConsumer(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(10000),
			},
			{
				ResourceName:  "apisix_consumer_plugins.quota",
				ImportState:   true,
				ImportStateId: testConsumer,
				ExpectError:   regexp.MustCompile("Invalid import ID"),
			},
		},
	})
}

// TestAccConsumerPlugins_missingConsumer asserts a clear error rather than a
// confusing 404 when the owning system has not created the consumer yet.
func TestAccConsumerPlugins_missingConsumer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "apisix_consumer_plugins" "missing" {
  consumer_id = "tf_acc_consumer_plugins_does_not_exist"

  plugins = {
    "limit-count" = jsonencode({ count = 1, time_window = 60 })
  }
}
`,
				ExpectError: regexp.MustCompile("Consumer does not exist"),
			},
		},
	})
}
