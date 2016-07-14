package softlayer

import (
	"fmt"
	"strconv"
	"testing"

	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSoftLayerScalePolicy_Basic(t *testing.T) {
	var scalepolicy datatypes.SoftLayer_Scale_Policy

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config:  testAccCheckSoftLayerScalePolicyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSoftLayerScalePolicyExists("softlayer_scale_policy.sample-http-cluster-policy", &scalepolicy),
					testAccCheckSoftLayerScalePolicyAttributes(&scalepolicy),
					resource.TestCheckResourceAttr(
						"softlayer_scale_policy.sample-http-cluster-policy", "name", "sample-http-cluster-policy"),
					resource.TestCheckResourceAttr(
						"softlayer_scale_policy.sample-http-cluster-policy", "scale_type", "RELATIVE"),
					resource.TestCheckResourceAttr(
						"softlayer_scale_policy.sample-http-cluster-policy", "scale_amount", "1"),
					resource.TestCheckResourceAttr(
						"softlayer_scale_policy.sample-http-cluster-policy", "cooldown", "30"),
					resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "triggers.#", "3"),
                                        testAccCheckSoftLayerScalePolicyContainsRepeatingTriggers(&scalepolicy, 2, "0 1 ? * MON,WED *"),
                                        resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "triggers.#", "3"),
                                        testAccCheckSoftLayerScalePolicyContainsResourceUseTriggers(&scalepolicy, 120, "80"),
				),
			},
			
			resource.TestStep{
                                Config: testAccCheckSoftLayerScalePolicyConfig_updated,
                                Check: resource.ComposeTestCheckFunc(
                                        testAccCheckSoftLayerScalePolicyExists("softlayer_scale_policy.sample-http-cluster-policy", &scalepolicy),
                                        resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "name", "changed-name"),
                                        resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "scale_amount", "2"),
                                        resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "cooldown", "35"),
                                        resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "triggers.#", "2"),
                                        testAccCheckSoftLayerScalePolicyContainsRepeatingTriggers(&scalepolicy, 2, "0 1 ? * MON,WED,SAT *"),
                                        resource.TestCheckResourceAttr(
                                                "softlayer_scale_policy.sample-http-cluster-policy", "triggers.#", "2"),
                                        testAccCheckSoftLayerScalePolicyContainsResourceUseTriggers(&scalepolicy, 130, "90"),
                                ),
                        },
		},
	})
}

func testAccCheckSoftLayerScalePolicyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client).scalePolicyService

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "softlayer_scale_policy" {
			continue
		}

		scalepolicyId, _ := strconv.Atoi(rs.Primary.ID)

		// Try to find the key
		_, err := client.GetObject(scalepolicyId)

		if err != nil {
			return fmt.Errorf("Waiting for Auto Scale Policy (%s) to be destroyed: %s", rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckSoftLayerScalePolicyContainsResourceUseTriggers(scalePolicy *datatypes.SoftLayer_Scale_Policy, period int, value string) resource.TestCheckFunc {
        return func(s *terraform.State) error {
                found := false

                for _, scaleResourceUseTrigger := range scalePolicy.ResourceUseTriggers {
                        for _, scaleResourceUseWatch := range scaleResourceUseTrigger.Watches {
                                if scaleResourceUseWatch.Metric == "host.cpu.percent" && scaleResourceUseWatch.Operator == ">" &&
                                        scaleResourceUseWatch.Period == period && scaleResourceUseWatch.Value == value {
                                        found = true
                                        break
                                }
                        }
                }

                if !found {
                        return fmt.Errorf("Resource use trigger not found in scale policy")

                }

                return nil
        }
}

func testAccCheckSoftLayerScalePolicyContainsRepeatingTriggers(scalePolicy *datatypes.SoftLayer_Scale_Policy, typeId int, schedule string) resource.TestCheckFunc {
        return func(s *terraform.State) error {
                found := false

                for _, scaleRepeatingTrigger := range scalePolicy.RepeatingTriggers {
                        if scaleRepeatingTrigger.TypeId == typeId && scaleRepeatingTrigger.Schedule == schedule {
                                found = true
                                break
                        }
                }

                if !found {
                        return fmt.Errorf("Repeating trigger %d with schedule %s not found in scale policy", typeId, schedule)

                }

                return nil
        }
}

func testAccCheckSoftLayerScalePolicyAttributes(scalepolicy *datatypes.SoftLayer_Scale_Policy) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if scalepolicy.Name != "sample-http-cluster-policy" {
			return fmt.Errorf("Bad name: %s", scalepolicy.Name)
		}

		return nil
	}
}

func testAccCheckSoftLayerScalePolicyExists(n string, scalepolicy *datatypes.SoftLayer_Scale_Policy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		scalepolicyId, _ := strconv.Atoi(rs.Primary.ID)

		client := testAccProvider.Meta().(*Client).scalePolicyService
		foundScalePolicy, err := client.GetObject(scalepolicyId)

		if err != nil {
			return err
		}

		if strconv.Itoa(int(foundScalePolicy.Id)) != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}

		*scalepolicy = foundScalePolicy

		return nil
	}
}

const testAccCheckSoftLayerScalePolicyConfig_basic = `
resource "softlayer_scale_group" "sample-http-cluster" {
    name = "sample-http-cluster"
    regional_group = "as-sgp-central-1" 
    cooldown = 30
    minimum_member_count = 1
    maximum_member_count = 10
    termination_policy = "CLOSEST_TO_NEXT_CHARGE"
    virtual_server_id = 267513
    port = 8080
    health_check = {
        type = "HTTP"
    }
    virtual_guest_member_template = {
        name = "test-VM"
        domain = "example.com"
        cpu = 1
        ram = 4096
        public_network_speed = 1000
        hourly_billing = true
        image = "DEBIAN_7_64"
        local_disk = false
        #disks = [25,100]
        region = "sng01"
        #post_install_script_uri = ""
        #ssh_keys = [383111]
        #user_data = "#!/bin/bash"
    }
    #network_vlans = {
    #    vlan_number = "1928"
    #    primary_router_hostname = "bcr02a.sng01"
    #}
 
}

resource "softlayer_scale_policy" "sample-http-cluster-policy" {
    name = "sample-http-cluster-policy"
    scale_type = "RELATIVE"
    scale_amount = 1
    cooldown = 30
    scale_group_id = "${softlayer_scale_group.sample-http-cluster.id}"
    triggers = {
        type = "RESOURCE_USE"
        watches = {

                    metric = "host.cpu.percent"
                    operator = ">"
                    value = "80"
                    period = 120
        }
    }
    triggers = {
        type = "ONE_TIME"
        date = "2016-07-30T23:55:00-00:00"
    }
    triggers = {
        type = "REPEATING"
        schedule = "0 1 ? * MON,WED *"
    }
    
}`

const testAccCheckSoftLayerScalePolicyConfig_updated = `
resource "softlayer_scale_group" "sample-http-cluster" {
    name = "sample-http-cluster"
    regional_group = "as-sgp-central-1"
    cooldown = 30
    minimum_member_count = 1
    maximum_member_count = 10
    termination_policy = "CLOSEST_TO_NEXT_CHARGE"
    virtual_server_id = 267513
    port = 8080
    health_check = {
        type = "HTTP"
    }
    virtual_guest_member_template = {
        name = "test-VM"
        domain = "example.com"
        cpu = 1
        ram = 4096
        public_network_speed = 1000
        hourly_billing = true
        image = "DEBIAN_7_64"
        local_disk = false
        #disks = [25,100]
        region = "sng01"
        #post_install_script_uri = ""
        #ssh_keys = [383111]
        #user_data = "#!/bin/bash"
    }
    #network_vlans = {
    #    vlan_number = "1928"
    #    primary_router_hostname = "bcr02a.sng01"
    #}

}
resource "softlayer_scale_policy" "sample-http-cluster-policy" {
    name = "changed-name"
    scale_type = "RELATIVE"
    scale_amount = 2
    cooldown = 35
    scale_group_id = "${softlayer_scale_group.sample-http-cluster.id}"
    triggers = {
        type = "RESOURCE_USE"
        watches = {

                    metric = "host.cpu.percent"
                    operator = ">"
                    value = "90"
                    period = 130
        }
    }
    triggers = {
        type = "REPEATING"
        schedule = "0 1 ? * MON,WED,SAT *"
    }

}`
