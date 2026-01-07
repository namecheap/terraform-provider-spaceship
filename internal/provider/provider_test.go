package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"terraform-provider-spaceship/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"spaceship": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6WithError(New("test")())()
	},
}

const testAccDefaultDomain = "dmytrovovk.com"

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("SPACESHIP_API_KEY") == "" {
		t.Skip("SPACESHIP_API_KEY must be set for acceptance testing")
	}

	if os.Getenv("SPACESHIP_API_SECRET") == "" {
		t.Skip("SPACESHIP_API_SECRET must be set for acceptance testing")
	}
}

func testAccDomainValue() string {
	if domain := os.Getenv("SPACESHIP_TEST_DOMAIN"); domain != "" {
		return domain
	}

	return testAccDefaultDomain

}

func testAccClient() (*client.Client, error) {
	return client.NewClient(
		defaultBaseURL,
		os.Getenv("SPACESHIP_API_KEY"),
		os.Getenv("SPACESHIP_API_SECRET"),
	)
}

func testAccCheckDNSRecordAbsent(domain, recordType, name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		testClient, err := testAccClient()
		if err != nil {
			return err
		}
		records, err := testClient.GetDNSRecords(context.Background(), domain)
		if err != nil {
			if client.IsNotFoundError(err) {
				return nil
			}
			return err
		}

		for _, record := range records {
			if strings.EqualFold(record.Type, recordType) && strings.EqualFold(record.Name, name) {
				return fmt.Errorf("DNS record %s %s still present in domain %s", record.Type, record.Name, domain)
			}
		}
		return nil
	}
}
